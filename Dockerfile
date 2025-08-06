# Multi-stage build for production-ready container
FROM golang:1.21-alpine AS builder

# Install security updates and required packages
RUN apk update && apk upgrade && apk add --no-cache \
    git \
    ca-certificates \
    tzdata && \
    update-ca-certificates

# Create appuser for security
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o server ./cmd/server

# Production stage
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

# Copy the binary
COPY --from=builder /app/server /server

# Copy migrations
COPY --from=builder /app/migrations /migrations

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/server", "-health-check"]

# Run the server
ENTRYPOINT ["/server"]