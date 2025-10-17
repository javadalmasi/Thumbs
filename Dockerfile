FROM golang:alpine3.21 AS build

WORKDIR /app/
RUN go env -w GOMODCACHE=/root/.cache/go-build

RUN apk add --no-cache build-base git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Vendor dependencies to ensure reproducible builds
RUN go mod vendor

# Verify the directory exists before building
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -mod=vendor -ldflags "-s -w -X 'main.version=build-$(date +%Y%m%d)'" -o Thumbs ./cmd/thumbs-server

FROM alpine:3.21

RUN adduser -u 10001 -S appuser

WORKDIR /app/

COPY --from=build /app/Thumbs /app/Thumbs

# Switch to non-privileged user
USER appuser

ENTRYPOINT ["/app/Thumbs"]