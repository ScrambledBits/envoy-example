# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum* ./

# Download dependencies (cached if go.mod/go.sum haven't changed)
RUN go mod download

# Copy source code
COPY main.go ./

# Build the application
# CGO_ENABLED=0 for static binary
# -ldflags="-w -s" to strip debug information and reduce binary size
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o web-service \
    main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies for health checks
RUN apk add --no-cache ca-certificates wget tzdata && \
    addgroup -g 65534 -S nonroot && \
    adduser -u 65534 -S nonroot -G nonroot

# Copy the binary from builder
COPY --from=builder /build/web-service /web-service

# Use non-root user for security
USER nonroot:nonroot

EXPOSE 1337

ENTRYPOINT ["/web-service"]
