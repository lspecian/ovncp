# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies with retry logic
RUN for i in 1 2 3; do \
        echo "Attempt $i: Downloading Go modules..." && \
        go mod download && break || \
        (echo "Attempt $i failed, waiting 15 seconds..." && sleep 15); \
    done && \
    echo "Go modules downloaded successfully"

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ovncp ./cmd/api

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S ovncp && \
    adduser -u 1000 -S ovncp -G ovncp

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/ovncp .
COPY --from=builder /app/migrations ./migrations

# Change ownership
RUN chown -R ovncp:ovncp /app

# Switch to non-root user
USER ovncp

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./ovncp"]