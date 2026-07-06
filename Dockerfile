# Stage 1: Build Go binary
FROM golang:1.25-bookworm@sha256:e17419604b6d1f9bc245694425f0ec9b1b53685c80850900a376fb10cb0f70cb AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Stage 2: Final image with texlive
FROM texlive/texlive:latest@sha256:20fa06486e76ebc91b93422153e22d8bca7d4e5366f4baf11693feea47d7ecde

# Install dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    git \
    && rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/server /app/server

# Expose port
EXPOSE 8080

# Set environment variables
ENV STATE_DIR=/data/repos
ENV PORT=8080

# Create data directory
RUN mkdir -p /data/repos

# Set entrypoint
ENTRYPOINT ["/app/server"]
