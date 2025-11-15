FROM texlive/texlive:latest

# Install dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    git \
    && rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy backend binary (to be built before docker build)
COPY server /app/server

# Expose port
EXPOSE 8080

# Set environment variables
ENV STATE_DIR=/data/repos
ENV PORT=8080

# Create data directory
RUN mkdir -p /data/repos

# Run the server
ENTRYPOINT ["/app/server"]
