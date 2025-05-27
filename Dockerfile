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

# Generate version file and build the binaries with static linking
RUN go generate ./cmd/agent && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.Version=$$(cat cmd/agent/version.txt)" -o bin/agent ./cmd/agent && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bin/tui ./cmd/tui

# Runtime stage - scratch for minimal image
FROM scratch

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binaries from builder
COPY --from=builder /app/bin/agent /bin/agent
COPY --from=builder /app/bin/tui /bin/tui

# Expose ports
EXPOSE 8080 7946 9090

# Default command
ENTRYPOINT ["/bin/agent"]
