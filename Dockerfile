# Multi-stage build for development and production

# Development stage
FROM golang:1.25-alpine AS development

# Install development tools
RUN apk add --no-cache \
    git \
    curl \
    bash \
    make \
    gcc \
    musl-dev \
    binutils-gold

# Copy Makefile first to leverage layer caching
COPY Makefile ./

# Install development tools using Makefile
RUN make install-tools

# Set working directory
WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Default command for development
CMD ["bash"]

# Production stage (for building the library)
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Run tests
RUN go test ./...

# Build (if there were a binary to build)
# RUN go build -o app .

# Final stage (minimal image for distribution)
FROM alpine:latest AS production

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy from builder stage
# COPY --from=builder /build/app .

# Since this is a library, we don't need a binary
# This stage could be used for documentation or examples

CMD ["sh"]