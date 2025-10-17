# Thumbs - Image Proxy Server

A high-performance, lightweight proxy server specifically designed for serving web images with enhanced functionality including quality detection, resizing, quality adjustment, format conversion, and encrypted ID support.

## Features

- **Fast Performance**: Uses HTTP/1.1, HTTP/2, and HTTP/3 clients for optimal performance
- **Quality Detection**: Automatically detects and serves the highest available quality image from any source
- **Image Parameters**: Supports resize, quality, and format conversion parameters
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
go build -mod=vendor -o Thumbs ./cmd/Thumbs
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

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `x-oss-process` | string | Alibaba-style image processing (e.g., `image/resize,m_fill,w_800,h_600`) | None |
| `resize` | string | Resize image to specified dimensions (width,height) | None |
| `quality` | integer | Image quality (1-100) | 85 |
| `format` | string | Output format (jpg, webp, avif) | webp |

#### Examples

```
# Get best quality thumbnail (12-character encoded ID)
/vi/2r8RVAuxuMN_

# Resize to 768x432 with 85% quality in webp format
/vi/2r8RVAuxuMN_?resize=768,432&quality=85

# Resize to 1280x720 with 90% quality in avif format
/vi/2r8RVAuxuMN_?resize=1280,720&quality=90&format=avif

# Get in jpg format with default quality
/vi/2r8RVAuxuMN_?format=jpg

# Alibaba-style resize (scale to 800x600)
/vi/2r8RVAuxuMN_?x-oss-process=image/resize,m_fill,w_800,h_600

# Full example with quality, format, and resize
/vi/2r8RVAuxuMN_?x-oss-process=image/resize,m_fill,w_1024,h_768&quality=90&format=webp
```

#### Alibaba-Style Image Processing

The proxy supports Alibaba Cloud Object Storage Service (OSS) style image processing parameters through the `x-oss-process` query parameter:

##### Resize Operations
- `x-oss-process=image/resize,m_fill,w_800,h_600` - Resize to 800x600 using fill mode
- `x-oss-process=image/resize,m_lfit,w_800,h_600` - Resize with limit fit (maintains aspect ratio, no upscale)
- `x-oss-process=image/resize,m_mfit,w_800,h_600` - Resize with manual fit (maintains aspect ratio, allows upscale)
- `x-oss-process=image/resize,m_pad,w_800,h_600` - Resize with padding to exact dimensions
- `x-oss-process=image/resize,w_800` - Resize width to 800, height auto-scaled
- `x-oss-process=image/resize,h_600` - Resize height to 600, width auto-scaled

##### Format Conversion
- `x-oss-process=image/format,jpg` - Convert to JPEG format
- `x-oss-process=image/format,png` - Convert to PNG format
- `x-oss-process=image/format,webp` - Convert to WebP format (served as JPEG in this implementation)
- `x-oss-process=image/format,avif` - Convert to AVIF format (served as JPEG in this implementation)

##### Quality Settings
- `x-oss-process=image/quality,q_90` - Set output quality to 90%

##### Combined Operations
- `x-oss-process=image/resize,w_800,h_600/format,jpg/quality,q_85` - Resize to 800x600, convert to JPEG, set quality to 85%

This allows for seamless integration with systems that already use Alibaba OSS image processing syntax.

#### Format Support Limitations
- JPEG: Full support for encoding and decoding
- PNG: Full support for encoding and decoding  
- WebP: Decoding supported, encoding available as JPEG in this implementation
- AVIF: Decoding supported, encoding available as JPEG in this implementation

For full WebP and AVIF encoding support, external tools would need to be integrated.

#### Demos
You can test the following endpoints with any encoded ID:

1. **Basic thumbnail:** `http://localhost:8080/vi/{encodedId}`
2. **Alibaba-style resize:** `http://localhost:8080/vi/{encodedId}?x-oss-process=image/resize,w_800,h_600`
3. **Quality adjustment:** `http://localhost:8080/vi/{encodedId}?x-oss-process=image/quality,q_90`
4. **Format conversion:** `http://localhost:8080/vi/{encodedId}?x-oss-process=image/format,jpg`
5. **Combined operations:** `http://localhost:8080/vi/{encodedId}?x-oss-process=image/resize,w_1024,h_768/format,jpg/quality,q_85`

### Encoded IDs

The proxy supports 12-character encoded IDs that are securely transformed from 11-character source IDs using XOR encryption. To use this feature:

1. Set the `YTPROXY_SECRET_KEY` environment variable with your secret key
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
      - YTPROXY_SECRET_KEY=your-16-char-key
      - YTPROXY_PORT=8080
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
  -e YTPROXY_SECRET_KEY=your-16-char-key \
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

# Test with resize parameters
curl "http://localhost:8080/vi/ENCODED_ID_HERE?resize=320,240"

# Test with Alibaba-style resize
curl "http://localhost:8080/vi/ENCODED_ID_HERE?x-oss-process=image/resize,w_320,h_240"

# Test with format and quality
curl "http://localhost:8080/vi/ENCODED_ID_HERE?format=jpg&quality=90"
```

### Testing Image Output

You can save and verify image properties:

```bash
# Download a thumbnail
curl -o test.jpg "http://localhost:8080/vi/ENCODED_ID_HERE?resize=320,240"

# Check file size and format
file test.jpg
ls -la test.jpg
```

### Example with Real Source ID

To generate an encoded ID for testing, you would need to use the Encode function with a real source ID. For example, for the source ID "dQw4w9WgXcQ", you would encode it using the secret key to get a 12-character encoded ID.

## License

This project is licensed under the BSD 3-Clause License - see the [LICENSE](LICENSE) file for details.