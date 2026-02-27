# Build stage
FROM golang:1.25-alpine AS builder

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application (static binary for scratch image)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o slashvibeprs .

# Final stage - minimal scratch image
FROM scratch

# Copy the binary from builder
COPY --from=builder /build/slashvibeprs /slashvibeprs

# Copy CA certificates for HTTPS (required for Slack API calls)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/slashvibeprs"]
