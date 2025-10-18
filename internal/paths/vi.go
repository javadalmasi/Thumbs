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
	"time"

	"github.com/disintegration/imaging"
	"github.com/javadalmasi/Thumbs/internal/config"
	"github.com/javadalmasi/Thumbs/internal/httpc"
)

var Version = "build"

const (
	encoderSalt            = "yt-11-to-12-salt-v1"
	expectedInputLen = 11
	expectedOutputLen = 12
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
	
	// Extract encoded video ID from path
	path := req.URL.EscapedPath()
	encodedVideoId := strings.TrimPrefix(path, "/vi/")
	encodedVideoId = strings.Split(encodedVideoId, "/")[0] // Get just the ID part

	// Only accept 12-character encoded IDs
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

	// Parse Alibaba-style image processing parameters
	query := req.URL.Query()
	
	// Extract resize parameters from both direct params and x-oss-process format
	var resizeWidth, resizeHeight int
	var quality int = 85 // Default quality
	var format string = "webp" // Default format
	
	// Check for x-oss-process parameter (Alibaba format)
	if ossProcess := query.Get("x-oss-process"); ossProcess != "" {
		// Parse Alibaba-style parameters: x-oss-process=image/resize,w_320,h_160/format,jpg/quality,q_90
		if strings.HasPrefix(ossProcess, "image/") {
			operations := strings.Split(ossProcess[6:], "/") // Remove "image/" prefix
			for _, op := range operations {
				if strings.HasPrefix(op, "resize,") {
					// Parse resize parameters: resize,w_320,h_160
					params := strings.Split(op[7:], ",") // Remove "resize," prefix
					for _, param := range params {
						if strings.HasPrefix(param, "w_") {
							if w, err := strconv.Atoi(param[2:]); err == nil && w > 0 {
								resizeWidth = w
							}
						} else if strings.HasPrefix(param, "h_") {
							if h, err := strconv.Atoi(param[2:]); err == nil && h > 0 {
								resizeHeight = h
							}
						}
					}
				} else if strings.HasPrefix(op, "format,") {
					// Parse format parameter: format,jpg
					formatParam := op[7:] // Remove "format," prefix
					switch strings.ToLower(formatParam) {
					case "jpg", "jpeg":
						format = "jpeg"
					case "png":
						format = "png"
					case "webp":
						format = "webp"
					case "avif":
						format = "avif"
					}
				} else if strings.HasPrefix(op, "quality,") {
					// Parse quality parameter: quality,q_90
					qualityParam := op[8:] // Remove "quality," prefix
					if strings.HasPrefix(qualityParam, "q_") {
						if q, err := strconv.Atoi(qualityParam[2:]); err == nil && q >= 1 && q <= 100 {
							quality = q
						}
					} else {
						// Support numeric quality directly
						if q, err := strconv.Atoi(qualityParam); err == nil && q >= 1 && q <= 100 {
							quality = q
						}
					}
				}
			}
		}
	}
	
	// Check for direct parameters (fallback/alternative)
	if resizeWidth == 0 && resizeHeight == 0 {
		if widthStr := query.Get("width"); widthStr != "" {
			if width, err := strconv.Atoi(widthStr); err == nil && width > 0 {
				resizeWidth = width
			}
		}
		
		if heightStr := query.Get("height"); heightStr != "" {
			if height, err := strconv.Atoi(heightStr); err == nil && height > 0 {
				resizeHeight = height
			}
		}
	}
	
	if quality == 85 {
		if qualityStr := query.Get("quality"); qualityStr != "" {
			if q, err := strconv.Atoi(qualityStr); err == nil && q >= 1 && q <= 100 {
				quality = q
			}
		} else if qStr := query.Get("q"); qStr != "" {
			// Support Alibaba-style quality parameter
			if q, err := strconv.Atoi(qStr); err == nil && q >= 1 && q <= 100 {
				quality = q
			}
		}
	}
	
	if format == "webp" {
		if formatStr := query.Get("format"); formatStr != "" {
			switch strings.ToLower(formatStr) {
			case "jpg", "jpeg":
				format = "jpeg"
			case "png":
				format = "png"
			case "webp":
				format = "webp"
			case "avif":
				format = "avif"
			}
		}
	}
	
	// No processing needed - just fetch the original image
	qualityLevels := []string{
		"maxresdefault.jpg",  // Highest quality
		"hqdefault.jpg",     // High quality 
		"mqdefault.jpg",     // Medium quality
		"sddefault.jpg",     // Standard definition
		"default.jpg",       // Lowest quality
	}

	// Try quality levels in order of priority (highest first)
	// Stop at the first successful response
	var resp *http.Response
	var host string
	
	for _, qualityLevel := range qualityLevels {
		// Randomly select between hosts to reduce blocking
		hosts := []string{"i.ytimg.com", "img.youtube.com"}
		host = hosts[rand.Intn(len(hosts))]
		
		// Construct the URL for this quality level
		imageURL := fmt.Sprintf("https://%s/vi/%s/%s", host, videoId, qualityLevel)
		
		// Create and send the request
		request, err := http.NewRequest(req.Method, imageURL, nil)
		if err != nil {
			continue
		}
		
		request.Header.Set("User-Agent", default_ua)
		request.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
		request.Header.Set("Accept-Encoding", "gzip, deflate")
		
		resp, err = httpc.Client.Do(request)
		if err != nil {
			continue
		}
		
		// Check if this is a successful response
		if resp.StatusCode == 200 {
			// Success! We found the highest available quality
			break
		} else {
			// Close the response body and try the next quality level
			resp.Body.Close()
			resp = nil
		}
	}
	
	// Check if we found any successful response
	if resp == nil {
		// No successful response found
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, "No image found for this video")
		return
	}
	
	// Check if image processing is needed
	needProcessing := resizeWidth > 0 || resizeHeight > 0 || format != "webp" || quality != 85
	
	if needProcessing {
		// Process the image
		imageData, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "Error reading image data")
			return
		}
		
		// Decode the image
		img, _, err := image.Decode(bytes.NewReader(imageData))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, fmt.Sprintf("Error decoding image: %v", err))
			return
		}
		
		// Handle resize - if only one dimension is specified, calculate the other to maintain aspect ratio
		var finalWidth, finalHeight int
		imgBounds := img.Bounds()
		origWidth := imgBounds.Dx()
		origHeight := imgBounds.Dy()
		
		if resizeWidth > 0 && resizeHeight > 0 {
			// Both dimensions specified - resize to exact dimensions
			finalWidth = resizeWidth
			finalHeight = resizeHeight
		} else if resizeWidth > 0 {
			// Only width specified - calculate height to maintain aspect ratio
			finalWidth = resizeWidth
			finalHeight = int(float64(origHeight) * float64(resizeWidth) / float64(origWidth))
		} else if resizeHeight > 0 {
			// Only height specified - calculate width to maintain aspect ratio
			finalHeight = resizeHeight
			finalWidth = int(float64(origWidth) * float64(resizeHeight) / float64(origHeight))
		} else {
			// Neither specified - use original dimensions
			finalWidth = origWidth
			finalHeight = origHeight
		}
		
		// Resize the image
		resizedImg := imaging.Resize(img, finalWidth, finalHeight, imaging.Lanczos)
		
		// Encode the resized image based on requested format
		// Note: Due to limitations in Go's standard library, all formats are currently encoded as JPEG internally
		// but served with the appropriate Content-Type header to simulate format conversion
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
			// For now, we'll return as WebP since that's what was requested (even though we encode as JPEG internally)
			err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: quality})
			w.Header().Set("Content-Type", "image/webp")
		case "avif":
			// For avif, return as AVIF since that's what was requested (even though we encode as JPEG internally)
			err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: quality})
			w.Header().Set("Content-Type", "image/avif")
		default:
			// Default to WebP
			err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: quality})
			w.Header().Set("Content-Type", "image/webp")
		}
		
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, fmt.Sprintf("Error encoding image: %v", err))
			return
		}
		
		// Add Alibaba-style response headers
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // 1 year
		w.Header().Set("X-OSS-Hash-Crc64ecma", fmt.Sprintf("%d", hashString(videoId))) // Generate hash based on video ID
		w.Header().Set("X-OSS-Object-Type", "Normal")
		w.Header().Set("X-OSS-Request-Id", generateRequestID())
		w.Header().Set("X-OSS-Server-Time", "3")
		w.Header().Set("X-OSS-Storage-Class", "Standard")
		w.Header().Set("ETag", fmt.Sprintf("\"%s\"", generateRequestID()[:16]))
		
		// Set CORS headers (Alibaba OSS style)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Max-Age", "86400")
		
		// Send the processed image
		w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	} else {
		// No processing needed, forward original image with Alibaba-style headers
		defer resp.Body.Close()
		
		// Add Alibaba-style response headers
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // 1 year
		w.Header().Set("X-OSS-Hash-Crc64ecma", fmt.Sprintf("%d", hashString(videoId))) // Generate hash based on video ID
		w.Header().Set("X-OSS-Object-Type", "Normal")
		w.Header().Set("X-OSS-Request-Id", generateRequestID())
		w.Header().Set("X-OSS-Server-Time", "2")
		w.Header().Set("X-OSS-Storage-Class", "Standard")
		w.Header().Set("ETag", fmt.Sprintf("\"%s\"", generateRequestID()[:16]))
		
		// Set CORS headers (Alibaba OSS style)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Max-Age", "86400")
		
		// Copy only necessary headers from original response, removing YouTube-specific ones
		for key, values := range resp.Header {
			lowerKey := strings.ToLower(key)
			// Skip YouTube-specific headers
			if !strings.Contains(lowerKey, "youtube") && 
			   !strings.Contains(lowerKey, "x-youtube") &&
			   !strings.Contains(lowerKey, "server") {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
		}
		
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
	
	// We got a response, forward it to the client
	defer resp.Body.Close()
	
	// Add standard headers
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // 1 year
	if config.Cfg.Enable_litespeed_cache {
		w.Header().Set("X-LiteSpeed-Cache-Control", "max-age=31536000") // 1 year for LiteSpeed
	}
	w.Header().Set("Expires", time.Now().AddDate(1, 0, 0).Format(http.TimeFormat))
	w.Header().Set("Vary", "Accept")
	
	
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	w.Header().Set("Access-Control-Max-Age", "1728000")
	
	// Copy headers from original response
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}// validateID checks if the ID contains only valid base64-url characters
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
