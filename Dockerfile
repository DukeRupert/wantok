# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o wantok ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata sqlite

# Create non-root user
RUN addgroup -g 1000 wantok && \
    adduser -u 1000 -G wantok -s /bin/sh -D wantok

# Create data directory
RUN mkdir -p /app/data && chown -R wantok:wantok /app

# Copy binary from builder
COPY --from=builder /app/wantok /app/wantok

# Switch to non-root user
USER wantok

# Expose port
EXPOSE 8080

# Set default environment variables
ENV DATABASE_PATH=/app/data/wantok.db
ENV PORT=8080
ENV SECURE_COOKIES=true

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/login || exit 1

# Run the server
CMD ["/app/wantok"]
