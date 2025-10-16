# YouTube Image Proxy

A high-performance, lightweight proxy server specifically designed for serving YouTube thumbnail images with enhanced functionality including quality detection, resizing, quality adjustment, format conversion, and encrypted ID support.

## Features

- **Fast Performance**: Uses HTTP/1.1, HTTP/2, and HTTP/3 clients for optimal performance
- **Quality Detection**: Automatically detects and serves the highest available quality thumbnail for any YouTube video
- **Image Parameters**: Supports resize, quality, and format conversion parameters
- **Encrypted IDs**: Supports encoded 12-character IDs that are securely decoded to 11-character YouTube IDs
- **Concurrent Requests**: Finds the best quality image efficiently using concurrent requests
- **Multiple Protocols**: Supports HTTP/1.1, HTTP/2, and HTTP/3 for maximum compatibility
- **CORS Enabled**: Built-in CORS headers for web application integration

## Installation

### Prerequisites

- Go 1.24 or higher

### Build from Source

```bash
git clone <repository-url>
cd http3-ytproxy
go mod download
go build -o http3-ytproxy ./cmd/http3-ytproxy
```

## Usage

### Basic Server

```bash
./http3-ytproxy -p 8080
```

### With Custom Parameters

```bash
# Start server on custom port and host
./http3-ytproxy -l 0.0.0.0 -p 3000

# Enable HTTP/3
./http3-ytproxy -http-client-ver 3

# Use Unix socket instead of TCP
./http3-ytproxy -uds -s /tmp/youtube-proxy.sock
```

## API Endpoints

### Image Proxy
```
/vi/{videoId}
```

Returns the highest quality thumbnail available for the given YouTube video ID.

#### Query Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `resize` | string | Resize image to specified dimensions (width,height) | None |
| `quality` | integer | Image quality (1-100) | 85 |
| `format` | string | Output format (jpg, webp, avif) | webp |

#### Examples

```
# Get best quality thumbnail (11-character YouTube ID)
/vi/dQw4w9WgXcQ

# Get best quality thumbnail (12-character encoded ID)
/vi/2r8RVAuxuMN_

# Resize to 768x432 with 85% quality in webp format
/vi/dQw4w9WgXcQ?resize=768,432&quality=85

# Resize to 1280x720 with 90% quality in avif format (using encoded ID)
/vi/2r8RVAuxuMN_?resize=1280,720&quality=90&format=avif

# Get in jpg format with default quality
/vi/dQw4w9WgXcQ?format=jpg
```

### Encoded IDs

The proxy supports 12-character encoded IDs that are securely transformed from 11-character YouTube video IDs using XOR encryption. To use this feature:

1. Set the `YTPROXY_SECRET_KEY` environment variable with your secret key
2. Use 12-character encoded IDs in place of the standard YouTube IDs
3. The proxy will automatically decode the ID and fetch the appropriate thumbnail

The encoding uses a deterministic, reversible algorithm:
- Input: 11-character YouTube ID using base64-url alphabet
- Output: 12-character encoded ID using base64-url alphabet
- The transformation is secured with your secret key using SHA256-derived 72-bit key

## Configuration

The proxy can be configured using command line flags or environment variables:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-p` | `YTPROXY_PORT` | `8080` | Port to listen on |
| `-l` | `YTPROXY_HOST` | `0.0.0.0` | Host to listen on |
| `-http` | `YTPROXY_ENABLE_HTTP` | `true` | Enable HTTP server |
| `-uds` | `YTPROXY_ENABLE_UDS` | `false` | Enable Unix Domain Socket |
| `-s` | `YTPROXY_UDS_PATH` | `/tmp/http-ytproxy.sock` | Unix socket path |
| `-http-client-ver` | `YTPROXY_HTTP_CLIENT_VER` | `1` | HTTP client version (1, 2, or 3) |
| `-ipv6-only` | `YTPROXY_IPV6_ONLY` | `false` | Use IPv6 only |
| `-pr` | `YTPROXY_PROXY` | `` | Proxy server to use |
| | `YTPROXY_SECRET_KEY` | `` | Secret key for ID encoding/decoding |

## Configuration

The proxy supports configuration via environment variables or a `.env` file. Create a `.env` file in the project root with the following format:

```
YTPROXY_SECRET_KEY=your-secret-key-here
YTPROXY_PORT=8080
YTPROXY_HOST=0.0.0.0
# Add other configuration variables as needed
```

## How It Works

1. When a request is made to `/vi/{videoId}`, the proxy extracts the video ID
2. Concurrently requests multiple quality versions of the thumbnail:
   - `maxresdefault.jpg`
   - `sddefault.jpg`
   - `mqdefault.jpg`
   - `hqdefault.jpg`
   - `default.jpg`
3. Returns the first successful response (highest available quality)
4. If transformation parameters are provided, applies them to the highest quality source

## Performance

The proxy uses concurrent requests to find the best quality image quickly, typically in less than 200ms depending on network conditions. It includes built-in connection management and supports HTTP/3 for maximum performance.

## Security

- Implements rate limiting (implicit through connection limits)
- Validates input parameters
- Sets appropriate security headers
- CORS enabled with permissive settings (can be configured)

## License

This project is licensed under the terms specified in the LICENSE file.