# Thumbs - Image Proxy Server

A high-performance, lightweight proxy server specifically designed for serving web images with enhanced functionality including quality detection, encrypted ID support, and concurrent requests.

## Features

- **Fast Performance**: Uses HTTP/1.1, HTTP/2, and HTTP/3 clients for optimal performance
- **Quality Detection**: Automatically detects and serves the highest available quality image from any source
- **Encrypted IDs**: Supports encoded 12-character IDs that are securely decoded to 11-character source IDs
- **Concurrent Requests**: Finds the best quality image efficiently using concurrent requests
- **Multiple Protocols**: Supports HTTP/1.1, HTTP/2, and HTTP/3 for maximum compatibility
- **CORS Enabled**: Built-in CORS headers for web application integration

## Installation

### Prerequisites

- Go 1.24 or higher

### Build from Source

```bash
git clone https://github.com/javadalmasi/Thumbs.git
cd Thumbs
go mod download
go build -mod=vendor -o Thumbs ./cmd/thumbs-server
```

## Usage

### Basic Server

```bash
./Thumbs -p 8080
```

### With Custom Parameters

```bash
# Start server on custom port and host
./Thumbs -l 0.0.0.0 -p 3000

# Enable HTTP/3
./Thumbs -http-client-ver 3

# Use Unix socket instead of TCP
./Thumbs -uds -s /tmp/thumbnail-proxy.sock
```

## API Endpoints

### Image Proxy
```
/vi/{encodedVideoId}
```

Returns the highest quality image available for the given encoded ID.

#### Query Parameters

All query parameters are ignored. The service retrieves and returns the highest quality image directly from YouTube without any processing.

#### Response Headers

The service returns various response headers, including caching headers. By default, the `X-LiteSpeed-Cache-Control` header is disabled. To enable it, set the `ENABLE_LITESPEED_CACHE` environment variable to `true`.

When enabled, the service will return the following cache headers:
- `Cache-Control`: `public, max-age=31536000, immutable` (1 year)
- `X-LiteSpeed-Cache-Control`: `max-age=31536000` (1 year, only when enabled)
- `Expires`: Set to 1 year from the request time

#### Examples

```
# Get best quality thumbnail (12-character encoded ID)
/vi/2r8RVAuxuMN_
```

#### Demos
You can test the following endpoint with any encoded ID:

1. **Basic thumbnail:** `http://localhost:8080/vi/{encodedId}`

### Encoded IDs

The proxy supports 12-character encoded IDs that are securely transformed from 11-character source IDs using XOR encryption. To use this feature:

1. Set the `SECRET_KEY` environment variable with your 16-character secret key
2. Use 12-character encoded IDs in place of the standard source IDs
3. The proxy will automatically decode the ID and fetch the appropriate image

The encoding uses a deterministic, reversible algorithm:
- Input: 11-character source ID using base64-url alphabet
- Output: 12-character encoded ID using base64-url alphabet
- The transformation is secured with your secret key using SHA256-derived 72-bit key

## Configuration

The proxy can be configured using command line flags or environment variables:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-p` | `PORT` | `8080` | Port to listen on |
| `-l` | `HOST` | `0.0.0.0` | Host to listen on |
| `-http` | `ENABLE_HTTP` | `true` | Enable HTTP server |
| `-uds` | `ENABLE_UDS` | `false` | Enable Unix Domain Socket |
| `-s` | `UDS_PATH` | `/tmp/http-ytproxy.sock` | Unix socket path |
| `-http-client-ver` | `HTTP_CLIENT_VER` | `1` | HTTP client version (1, 2, or 3) |
| `-ipv6-only` | `IPV6_ONLY` | `false` | Use IPv6 only |
| `-pr` | `PROXY` | `` | Proxy server to use |
| | `SECRET_KEY` | `` | Secret key for ID encoding/decoding (exactly 16 characters) |
| | `ENABLE_LITESPEED_CACHE` | `false` | Enable X-LiteSpeed-Cache-Control header (set to `true` to enable) |

## Configuration

The proxy supports configuration via environment variables or a `.env` file. Create a `.env` file in the project root with the following format:

```
SECRET_KEY=your-16-char-secret-key  # Must be exactly 16 characters
PORT=8080
HOST=0.0.0.0
ENABLE_LITESPEED_CACHE=true  # Set to true to enable X-LiteSpeed-Cache header
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

## Docker Deployment

The application is available as a Docker container on GitHub Container Registry (GHCR).

### Pull from GHCR

```bash
docker pull ghcr.io/javadalmasi/thumbs:latest
```

### Docker Compose

Create a `docker-compose.yml` file:

```yaml
# This is already configured in the project's docker-compose.yml
services:
  Thumbs:
    build: .
    image: ghcr.io/javadalmasi/thumbs:latest
    container_name: Thumbs
    restart: unless-stopped
    ports:
      - "8080:8080/tcp" # HTTP
    environment:
      - SECRET_KEY=your-16-char-key  # Must be exactly 16 characters
      - PORT=8080
    cap_add:
      - NET_ADMIN

networks:
  thumbs_network:
    driver: bridge
```

### Building and Running with Docker

```bash
# Build the image
docker build -t ghcr.io/javadalmasi/thumbs:latest .

# Run the container
docker run -d \
  --name Thumbs \
  -p 8080:8080 \
  -e SECRET_KEY=your-16-char-key  # Must be exactly 16 characters \
  --cap-add=NET_ADMIN \
  ghcr.io/javadalmasi/thumbs:latest
```

## Testing

### Manual Testing

After starting the server, you can test it with curl:

```bash
# Test the root endpoint
curl http://localhost:8080

# Test with an encoded ID (12-character encoded ID)
curl http://localhost:8080/vi/ENCODED_ID_HERE
```

### Testing Image Output

You can save and verify image properties:

```bash
# Download a thumbnail
curl -o test.jpg "http://localhost:8080/vi/ENCODED_ID_HERE"

# Check file size and format
file test.jpg
ls -la test.jpg
```

### Example with Real Source ID

To generate an encoded ID for testing, you would need to use the Encode function with a real source ID. For example, for the source ID "dQw4w9WgXcQ", you would encode it using the secret key to get a 12-character encoded ID.

## License

This project is licensed under the BSD 3-Clause License - see the [LICENSE](LICENSE) file for details.