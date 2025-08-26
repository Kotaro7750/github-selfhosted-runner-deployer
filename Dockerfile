# Build stage
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Install ca-certificates in build stage
RUN apk --no-cache add ca-certificates

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o github-selfhosted-runner-deployer .

# Runtime stage
FROM scratch

# Copy ca-certificates from build stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from build stage
COPY --from=builder /app/github-selfhosted-runner-deployer /github-selfhosted-runner-deployer

# Run the application
ENTRYPOINT ["/github-selfhosted-runner-deployer"]
