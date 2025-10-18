# LiteSpeed Cache Control Feature

## Overview

The Thumbs service includes an optional `X-LiteSpeed-Cache-Control` header that can be enabled to provide additional caching control for LiteSpeed web servers. By default, this feature is **disabled**.

## Enabling the Feature

To enable the `X-LiteSpeed-Cache-Control` header, set the environment variable `ENABLE_LITESPEED_CACHE` to `true`.

### Using Environment Variable

```bash
export ENABLE_LITESPEED_CACHE=true
```

### Using Docker

```bash
docker run -d \
  --name Thumbs \
  -p 8080:8080 \
  -e SECRET_KEY=your-16-char-key \
  -e ENABLE_LITESPEED_CACHE=true \
  --cap-add=NET_ADMIN \
  ghcr.io/javadalmasi/thumbs:latest
```

### Using Docker Compose

Update your `docker-compose.yml`:

```yaml
services:
  Thumbs:
    build: .
    image: ghcr.io/javadalmasi/thumbs:latest
    container_name: Thumbs
    restart: unless-stopped
    ports:
      - "8080:8080/tcp"
    environment:
      - SECRET_KEY=your-16-char-key  # Must be exactly 16 characters
      - ENABLE_LITESPEED_CACHE=true  # Enable LiteSpeed cache header
    cap_add:
      - NET_ADMIN
    networks:
      - thumbs_network
```

### Using .env File

Add the variable to your `.env` file:

```
SECRET_KEY=your-16-char-secret-key  # Must be exactly 16 characters
YTPROXY_PORT=8080
YTPROXY_HOST=0.0.0.0
ENABLE_LITESPEED_CACHE=true  # Set to true to enable X-LiteSpeed-Cache header
```

## Behavior

- **Default (Disabled)**: Only standard `Cache-Control` header is sent
- **Enabled**: Both `Cache-Control` and `X-LiteSpeed-Cache-Control` headers are sent

### Headers When Disabled:
```
Cache-Control: public, max-age=31536000, immutable
Expires: [future date, 1 year from request]
```

### Headers When Enabled:
```
Cache-Control: public, max-age=31536000, immutable
X-LiteSpeed-Cache-Control: max-age=31536000
Expires: [future date, 1 year from request]
```

## Feature Details

- The header is only added when image transformations are applied (though currently no transformations are processed)
- The cache duration is set to 1 year (31536000 seconds) 
- The header helps LiteSpeed web servers cache processed images more efficiently
- When disabled (default), no additional LiteSpeed-specific header is added

## Configuration

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| | `ENABLE_LITESPEED_CACHE` | `false` | Enable X-LiteSpeed-Cache-Control header (set to `true` to enable) |