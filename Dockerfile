# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ovncp ./cmd/ovncp

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