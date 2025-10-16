package paths

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"git.nadeko.net/Fijxu/http3-ytproxy/internal/config"
	"git.nadeko.net/Fijxu/http3-ytproxy/internal/httpc"
)

func forbiddenChecker(resp *http.Response, w http.ResponseWriter) error {
	if resp.StatusCode == 403 {
		w.WriteHeader(403)
		return fmt.Errorf("forbidden")
	}
	return nil
}

func Vi(w http.ResponseWriter, req *http.Request) {
	const host string = "i.ytimg.com"
	
	// Extract encoded video ID from path
	path := req.URL.EscapedPath()
	encodedVideoId := strings.TrimPrefix(path, "/vi/")
	encodedVideoId = strings.Split(encodedVideoId, "/")[0] // Get just the ID part

	// Check if the ID is 12 characters (encoded) or 11 characters (raw YouTube ID)
	var videoId string
	if len(encodedVideoId) == 12 {
		// Decode the 12-character encoded ID to get the 11-character YouTube ID
		secret := config.Cfg.Companion.Secret_key
		if secret == "" {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "Secret key not configured")
			return
		}
		
		decodedId, err := Decode(encodedVideoId, secret)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, fmt.Sprintf("Invalid encoded ID: %v", err))
			return
		}
		videoId = decodedId
	} else if len(encodedVideoId) == 11 {
		// It's already a raw YouTube ID
		videoId = encodedVideoId
	} else {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, fmt.Sprintf("Invalid ID length: got %d, expected 11 or 12", len(encodedVideoId)))
		return
	}

	// Parse query parameters for resize, quality, and format
	query := req.URL.Query()
	
	// Get resize parameters
	var resizeWidth, resizeHeight int
	if resizeParam := query.Get("resize"); resizeParam != "" {
		parts := strings.Split(resizeParam, ",")
		if len(parts) == 2 {
			if w, err := strconv.Atoi(parts[0]); err == nil {
				resizeWidth = w
			}
			if h, err := strconv.Atoi(parts[1]); err == nil {
				resizeHeight = h
			}
		}
	}
	
	// Get quality parameter (default 85)
	quality := 85
	if q := query.Get("quality"); q != "" {
		if qVal, err := strconv.Atoi(q); err == nil && qVal >= 1 && qVal <= 100 {
			quality = qVal
		}
	}
	
	// Get format parameter (default webp)
	format := "webp"
	if f := query.Get("format"); f != "" {
		if f == "jpg" || f == "jpeg" || f == "webp" || f == "avif" {
			format = f
		}
	}

	// If resize, quality, or format parameters are specified, 
	// we need to handle the transformation
	if resizeWidth > 0 && resizeHeight > 0 || quality != 85 || format != "webp" {
		// Define the quality levels in order of preference (highest to lowest)
		qualityLevels := []string{
			"maxresdefault.jpg",
			"sddefault.jpg", 
			"mqdefault.jpg",
			"hqdefault.jpg", 
			"default.jpg",
		}

		// Channel to receive the first successful response
		responseChan := make(chan *http.Response, 1)
		var wg sync.WaitGroup

		// Make concurrent requests for each quality level
		for _, qualityLevel := range qualityLevels {
			wg.Add(1)
			go func(qualityLevel string) {
				defer wg.Done()
				
				// Check if a response has already been found
				select {
				case <-responseChan:
					return // Another goroutine already found a response
				default:
				}
				
				// Construct the URL for this quality level
				imageURL := fmt.Sprintf("https://%s/vi/%s/%s", host, videoId, qualityLevel)
				
				// Create and send the request
				request, err := http.NewRequest(req.Method, imageURL, nil)
				if err != nil {
					return
				}
				
				request.Header.Set("User-Agent", default_ua)
				request.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
				request.Header.Set("Accept-Encoding", "gzip, deflate")
				
				resp, err := httpc.Client.Do(request)
				if err != nil {
					return
				}
				
				// Check if this is a successful response
				if resp.StatusCode == 200 {
					// Try to send the response, but only if channel is not full
					select {
					case responseChan <- resp:
					default:
						// Another goroutine already sent a response, close this one
						resp.Body.Close()
					}
				} else {
					// Close the response body if not successful
					resp.Body.Close()
				}
			}(qualityLevel)
		}
		
		// Wait for all goroutines to finish or for a response to be found
		go func() {
			wg.Wait()
			// Close the channel after all requests are done
			close(responseChan)
		}()
		
		// Get the first successful response
		resp, ok := <-responseChan
		if !ok {
			// No successful response found
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "No image found for this video")
			return
		}
		
		// We got a response, forward it to the client
		defer resp.Body.Close()
		
		// Set appropriate content type based on requested format
		switch format {
		case "jpg", "jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case "webp":
			w.Header().Set("Content-Type", "image/webp")
		case "avif":
			w.Header().Set("Content-Type", "image/avif")
		default:
			// Use the original content type from the response
			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
		}
		
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	} else {
		// No transformation parameters, use original logic for best quality image
		qualityLevels := []string{
			"maxresdefault.jpg",
			"sddefault.jpg", 
			"mqdefault.jpg",
			"hqdefault.jpg", 
			"default.jpg",
		}

		// Channel to receive the first successful response
		responseChan := make(chan *http.Response, 1)
		var wg sync.WaitGroup

		// Make concurrent requests for each quality level
		for _, qualityLevel := range qualityLevels {
			wg.Add(1)
			go func(qualityLevel string) {
				defer wg.Done()
				
				// Check if a response has already been found
				select {
				case <-responseChan:
					return // Another goroutine already found a response
				default:
				}
				
				// Construct the URL for this quality level
				imageURL := fmt.Sprintf("https://%s/vi/%s/%s", host, videoId, qualityLevel)
				
				// Create and send the request
				request, err := http.NewRequest(req.Method, imageURL, nil)
				if err != nil {
					return
				}
				
				request.Header.Set("User-Agent", default_ua)
				request.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
				request.Header.Set("Accept-Encoding", "gzip, deflate")
				
				resp, err := httpc.Client.Do(request)
				if err != nil {
					return
				}
				
				// Check if this is a successful response
				if resp.StatusCode == 200 {
					// Try to send the response, but only if channel is not full
					select {
					case responseChan <- resp:
					default:
						// Another goroutine already sent a response, close this one
						resp.Body.Close()
					}
				} else {
					// Close the response body if not successful
					resp.Body.Close()
				}
			}(qualityLevel)
		}
		
		// Wait for all goroutines to finish or for a response to be found
		go func() {
			wg.Wait()
			// Close the channel after all requests are done
			close(responseChan)
		}()
		
		// Get the first successful response
		resp, ok := <-responseChan
		if !ok {
			// No successful response found
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "No image found for this video")
			return
		}
		
		// We got a response, forward it to the client
		defer resp.Body.Close()
		
		// Copy headers from the response
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

const (
	encoderSalt            = "yt-11-to-12-salt-v1"
	expectedInputLen = 11
	expectedOutputLen = 12
)

// validateID checks if the ID contains only valid base64-url characters
func validateID(id string, expectedLen int) error {
	if len(id) != expectedLen {
		return fmt.Errorf("invalid length: expected %d, got %d", expectedLen, len(id))
	}

	for _, c := range id {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return fmt.Errorf("invalid character: %c", c)
		}
	}
	return nil
}

// deriveKey derives a 72-bit key from the secret and salt using SHA256
func deriveKey(secret string) (*big.Int, error) {
	h := sha256.New()
	h.Write([]byte(secret))
	h.Write([]byte(encoderSalt))
	
	// Use first 9 bytes (72 bits) of the hash
	derived := h.Sum(nil)[:9]
	
	// Convert to big.Int (MSB first)
	key := new(big.Int).SetBytes(derived)
	return key, nil
}

// Encode converts an 11-character YouTube ID to a 12-character encoded ID
func Encode(id11, secret string) (string, error) {
	if err := validateID(id11, expectedInputLen); err != nil {
		return "", fmt.Errorf("invalid input ID: %w", err)
	}

	// Decode the 11-character ID from base64-url to get the 66-bit value
	// Add proper padding for base64 decoding
	paddedInput := id11
	switch len(paddedInput) % 4 {
	case 2:
		paddedInput += "=="
	case 3:
		paddedInput += "="
	}
	
	decodedBytes, err := base64.URLEncoding.DecodeString(paddedInput)
	if err != nil {
		return "", fmt.Errorf("failed to decode input: %w", err)
	}

	// Create a 66-bit value as a big.Int
	// The decoded bytes represent 66 bits, so we shift them appropriately
	// in a 72-bit space (9 bytes)
	shiftedInput := make([]byte, 9) // 9 bytes = 72 bits
	copy(shiftedInput[9-len(decodedBytes):], decodedBytes)
	n := new(big.Int).SetBytes(shiftedInput)

	// Derive the key S (72 bits)
	s, err := deriveKey(secret)
	if err != nil {
		return "", fmt.Errorf("failed to derive key: %w", err)
	}

	// Compute M = N XOR S
	m := new(big.Int)
	m.Xor(n, s)
	
	// Convert M to 9 bytes (ensuring 72 bits)
	mBytes := m.Bytes()
	if len(mBytes) > 9 {
		mBytes = mBytes[len(mBytes)-9:] // Take least significant bytes
	}
	
	finalMBytes := make([]byte, 9)
	copy(finalMBytes[9-len(mBytes):], mBytes)
	
	// Encode to exactly 12 base64-url characters (9 bytes = 12 chars)
	result := base64.URLEncoding.EncodeToString(finalMBytes)
	
	return result[:12], nil
}

// Decode converts a 12-character encoded ID back to an 11-character YouTube ID
func Decode(id12, secret string) (string, error) {
	if err := validateID(id12, expectedOutputLen); err != nil {
		return "", fmt.Errorf("invalid input ID: %w", err)
	}

	// Decode the 12-character ID from base64-url (gives us 9 bytes = 72 bits)
	decodedBytes, err := base64.URLEncoding.DecodeString(id12)
	if err != nil {
		return "", fmt.Errorf("failed to decode input: %w", err)
	}

	// Convert to big integer M (72 bits)
	m := new(big.Int).SetBytes(decodedBytes)

	// Derive the same key S (72 bits)
	s, err := deriveKey(secret)
	if err != nil {
		return "", fmt.Errorf("failed to derive key: %w", err)
	}

	// Compute N = M XOR S
	n := new(big.Int)
	n.Xor(m, s)

	// Get the 66-bit value (from the 72-bit space)
	// We need to mask to 66 bits to ensure it fits in 11 base64-url characters
	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 66), big.NewInt(1)) // (2^66) - 1
	n66 := new(big.Int).And(n, mask)

	// Convert the 66-bit value back to bytes
	nBytes := n66.Bytes()
	
	// Calculate how many bytes represent our 66-bit value
	var relevantBytes []byte
	if len(nBytes) <= 9 {
		relevantBytes = nBytes
	} else {
		// Take the least significant 9 bytes if result is larger
		relevantBytes = nBytes[len(nBytes)-9:]
	}

	// Create proper base64 string for 66 bits (11 chars)
	tempEncoded := base64.URLEncoding.EncodeToString(relevantBytes)
	
	// Since 66 bits should encode to 11 characters, we take first 11 if available
	if len(tempEncoded) >= 11 {
		return tempEncoded[:11], nil
	} else {
		return tempEncoded, nil
	}
}
