package paths

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash/crc64"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math/big"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	"github.com/javadalmasi/Thumbs/internal/config"
	"github.com/javadalmasi/Thumbs/internal/httpc"
)

func forbiddenChecker(resp *http.Response, w http.ResponseWriter) error {
	if resp.StatusCode == 403 {
		w.WriteHeader(403)
		return fmt.Errorf("forbidden")
	}
	return nil
}

// Helper function to generate hash based on string input
func hashString(s string) uint64 {
	// Using simple FNV-1a hash algorithm
	h := crc64.MakeTable(crc64.ECMA) // Using crc64 as a hash function
	return crc64.Checksum([]byte(s), h)
}

// Helper function to generate request ID
func generateRequestID() string {
	// Generate a random request ID similar to Alibaba OSS
	rand.Seed(time.Now().UnixNano())
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 24)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func Vi(w http.ResponseWriter, req *http.Request) {
	const host string = "i.ytimg.com"
	
	// Extract encoded video ID from path
	path := req.URL.EscapedPath()
	encodedVideoId := strings.TrimPrefix(path, "/vi/")
	encodedVideoId = strings.Split(encodedVideoId, "/")[0] // Get just the ID part

	// Only accept 12-character encoded IDs, no more 11-character YouTube IDs
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
	} else {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, fmt.Sprintf("Invalid ID length: got %d, expected 12 for encoded ID", len(encodedVideoId)))
		return
	}

	// Parse query parameters for resize, quality, and format
	query := req.URL.Query()
	
	// Initialize default values
	var resizeWidth, resizeHeight int
	quality := 85
	format := "webp"
	
	// Parse only x-oss-process parameter for Alibaba-style processing
	ossProcessParam := query.Get("x-oss-process")
	if ossProcessParam != "" {
		// Parse x-oss-process=image/resize,w_800,h_600 or image/format,jpg or image/quality,q_85
		// Can also be combined like: image/resize,w_800,h_600/format,jpg/quality,q_90
		operations := strings.Split(ossProcessParam, "/")
		
		for _, operation := range operations {
			operation = strings.TrimSpace(operation)
			
			// Handle resize operations
			if strings.HasPrefix(operation, "image/resize") {
				parts := strings.Split(operation, ",")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if strings.HasPrefix(part, "w_") {
						if w, err := strconv.Atoi(strings.TrimPrefix(part, "w_")); err == nil {
							resizeWidth = w
						}
					} else if strings.HasPrefix(part, "h_") {
						if h, err := strconv.Atoi(strings.TrimPrefix(part, "h_")); err == nil {
							resizeHeight = h
						}
					}
				}
			} else if strings.HasPrefix(operation, "image/format") {
				// Handle format operations
				parts := strings.Split(operation, ",")
				if len(parts) >= 2 {
					newFormat := strings.TrimSpace(parts[1])
					if newFormat == "jpg" || newFormat == "jpeg" {
						format = "jpg"
					} else if newFormat == "png" {
						format = "png"
					} else if newFormat == "webp" {
						format = "webp"
					} else if newFormat == "avif" {
						format = "avif"
					}
				}
			} else if strings.HasPrefix(operation, "image/quality") {
				// Handle quality operations
				parts := strings.Split(operation, ",")
				if len(parts) >= 2 {
					qualityStr := strings.TrimPrefix(parts[1], "q_")
					if q, err := strconv.Atoi(qualityStr); err == nil && q >= 1 && q <= 100 {
						quality = q
					}
				}
			}
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
		
		// We got a response, process and forward it to the client
		defer resp.Body.Close()
		
		// Add cache headers for CDN and LiteSpeed
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // 1 year
		if config.Cfg.Enable_litespeed_cache {
			w.Header().Set("X-LiteSpeed-Cache-Control", "max-age=31536000") // 1 year for LiteSpeed
		}
		w.Header().Set("Expires", time.Now().AddDate(1, 0, 0).Format(http.TimeFormat))
		w.Header().Set("Vary", "Accept")
		
		// Add Alibaba-like response headers
		w.Header().Set("X-Bucket-Code", "3")
		w.Header().Set("X-OSS-Hash-Crc64Ecma", fmt.Sprintf("%d", hashString(videoId))) // Generate hash based on video ID
		w.Header().Set("X-OSS-Object-Type", "Normal")
		w.Header().Set("X-OSS-Request-ID", generateRequestID())
		w.Header().Set("X-OSS-Server-Time", "2")
		w.Header().Set("X-OSS-Storage-Class", "Standard")

		// Read the image data
		imageData, err := io.ReadAll(resp.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "Error reading image data")
			return
		}
		
		// Only process the image if resize parameters are specified
		if resizeWidth > 0 && resizeHeight > 0 {
			// Decode the image
			img, _, err := image.Decode(bytes.NewReader(imageData))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				io.WriteString(w, fmt.Sprintf("Error decoding image: %v", err))
				return
			}
			
			// Resize the image
			resizedImg := imaging.Resize(img, resizeWidth, resizeHeight, imaging.Lanczos)
			
			// Encode the resized image based on requested format
			var buf bytes.Buffer
			switch format {
			case "jpg", "jpeg":
				err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: quality})
				w.Header().Set("Content-Type", "image/jpeg")
			case "png":
				err = png.Encode(&buf, resizedImg)
				w.Header().Set("Content-Type", "image/png")
			case "webp":
				// For webp, we need to handle this separately as Go stdlib doesn't encode webp
				// For now, we'll return as JPEG since we don't have webp encoding
				err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: quality})
				w.Header().Set("Content-Type", "image/jpeg")
			case "avif":
				// For avif, return as JPEG since Go's standard library doesn't support encoding these natively
				// In a production environment, you might want to integrate with external tools like avifenc
				err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: quality})
				w.Header().Set("Content-Type", "image/jpeg")
			default:
				// Default to JPEG
				err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: quality})
				w.Header().Set("Content-Type", "image/jpeg")
			}
			
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				io.WriteString(w, fmt.Sprintf("Error encoding image: %v", err))
				return
			}
			
			// Send the processed image
			w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
			w.WriteHeader(resp.StatusCode)
			w.Write(buf.Bytes())
		} else {
			// No resize needed, but we may still need to convert quality/format
			if quality != 85 || format != "webp" {
				// Decode the image
				img, _, err := image.Decode(bytes.NewReader(imageData))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					io.WriteString(w, fmt.Sprintf("Error decoding image: %v", err))
					return
				}
				
				// Encode with the requested quality/format
				var buf bytes.Buffer
				switch format {
				case "jpg", "jpeg":
					err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
					w.Header().Set("Content-Type", "image/jpeg")
				case "png":
					err = png.Encode(&buf, img)
					w.Header().Set("Content-Type", "image/png")
				case "webp":
					// For webp, return as JPEG since Go's standard library doesn't support encoding these natively
					err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
					w.Header().Set("Content-Type", "image/jpeg")
				case "avif":
					// For avif, return as JPEG since Go doesn't support encoding these natively
					err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
					w.Header().Set("Content-Type", "image/jpeg")
				default:
					// Default to JPEG with requested quality
					err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
					w.Header().Set("Content-Type", "image/jpeg")
				}
				
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					io.WriteString(w, fmt.Sprintf("Error encoding image: %v", err))
					return
				}
				
				w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
				w.WriteHeader(resp.StatusCode)
				w.Write(buf.Bytes())
			} else {
				// No processing needed, forward original image with cache headers
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
				w.Write(imageData)
			}
		}
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

	// Derive the same key S (72 bits)
	s, err := deriveKey(secret)
	if err != nil {
		return "", fmt.Errorf("failed to derive key: %w", err)
	}

	// Convert to big integer M (72 bits)
	m := new(big.Int).SetBytes(decodedBytes)

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
	// Add proper padding for base64 encoding if necessary
	tempEncoded := base64.URLEncoding.EncodeToString(relevantBytes)
	
	// Remove padding and ensure exactly 11 characters
	// Since 66 bits should encode to 11 characters, we take first 11 if available
	if len(tempEncoded) >= 11 {
		// Remove any padding added during encoding
		result := tempEncoded[:11]
		return result, nil
	} else {
		// Pad with zeros if needed to ensure 11 characters
		result := tempEncoded
		for len(result) < 11 {
			result = "A" + result  // Use 'A' which represents zero in base64
		}
		return result[:11], nil
	}
}
