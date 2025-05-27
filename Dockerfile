# Build stage
FROM golang:1.24.3-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binaries
RUN make build

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl

# Create non-root user
RUN adduser -D -s /bin/sh llm

# Copy binaries from builder
COPY --from=builder /app/bin/agent /bin/agent
COPY --from=builder /app/bin/tui /bin/tui

# Create directories
RUN mkdir -p /models /config /logs && \
    chown -R llm:llm /models /config /logs

# Switch to non-root user for most operations
USER llm

# Expose ports
EXPOSE 8080 7946

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command
CMD ["/bin/agent"]
