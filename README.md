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

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `x-oss-process` | string | Alibaba-style image processing (e.g., `image/resize,m_fill,w_800,h_600`) | None |
| `resize` | string | Resize image to specified dimensions (width,height) | None |
| `quality` | integer | Image quality (1-100) | 85 |
| `format` | string | Output format (jpg, webp, avif) | webp |

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

The proxy supports Alibaba Cloud Object Storage Service (OSS) style image processing parameters through the `x-oss-process` query parameter. All operations use the format: `x-oss-process=image/{operation},{parameters}`

**Default Values:**
- Format: `webp` (default output format)
- Quality: `85` (default quality level)

##### Resize Operations
- `x-oss-process=image/resize,w_800,h_600` - Resize to 800x600 while maintaining aspect ratio
- `x-oss-process=image/resize,w_800` - Resize width to 800, height auto-scaled maintaining aspect ratio
- `x-oss-process=image/resize,h_600` - Resize height to 600, width auto-scaled maintaining aspect ratio

**Important:** Currently only width and height parameters are supported. Mode parameters (m_fill, m_lfit, etc.) are not implemented.

##### Format Conversion
Supported formats:
- `x-oss-process=image/format,jpg` - Convert to JPEG format
- `x-oss-process=image/format,png` - Convert to PNG format  
- `x-oss-process=image/format,webp` - Convert to WebP format (converted to JPEG in this implementation due to Go standard library limitations)
- `x-oss-process=image/format,avif` - Convert to AVIF format (converted to JPEG in this implementation due to Go standard library limitations)

##### Quality Settings
- `x-oss-process=image/quality,q_90` - Set output quality to 90% (range: 1-100)
- `x-oss-process=image/quality,q_75` - Set output quality to 75% (range: 1-100)

##### Combined Operations
Multiple operations can be combined by separating them with `/`:
- `x-oss-process=image/resize,w_800,h_600/format,jpg` - Resize to 800x600 and convert to JPEG
- `x-oss-process=image/format,png/quality,q_90` - Convert to PNG with 90% quality
- `x-oss-process=image/resize,w_1024,h_768/format,webp/quality,q_85` - Resize to 1024x768, convert to WebP, set quality to 85%

**Note:** When requesting formats that Go's standard library cannot encode (WebP, AVIF), the image will be converted to JPEG but with appropriate Content-Type headers.

##### Processing Trigger
Image processing is triggered when any of the following conditions are met:
- Resize parameters are specified (both width and height > 0)
- Quality is different from default (85)
- Format is different from default (webp)

If only the default format (webp) is requested without other transformations, no processing occurs and the original image is served.

##### Examples
- Basic resize: `x-oss-process=image/resize,w_320,h_240`
- Format conversion: `x-oss-process=image/format,jpg`  
- Quality adjustment: `x-oss-process=image/quality,q_90`
- Combined operations: `x-oss-process=image/resize,w_400,h_300/format,png/quality,q_75`

#### Demos
You can test the following endpoints with any encoded ID:

1. **Basic thumbnail:** `http://localhost:8080/vi/{encodedId}`
2. **Alibaba-style resize:** `http://localhost:8080/vi/{encodedId}?x-oss-process=image/resize,w_800,h_600`
3. **Quality adjustment:** `http://localhost:8080/vi/{encodedId}?x-oss-process=image/quality,q_90`
4. **Format conversion:** `http://localhost:8080/vi/{encodedId}?x-oss-process=image/format,jpg`
5. **Combined operations:** `http://localhost:8080/vi/{encodedId}?x-oss-process=image/resize,w_1024,h_768/format,jpg/quality,q_85`

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