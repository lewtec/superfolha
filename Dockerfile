# Runtime image for GoReleaser (dockers_v2).
# Binaries are built by GoReleaser and copied from the build context as
#   $TARGETPLATFORM/superfolha
# Do not rebuild Go/frontend here — see Dockerfile.build for a full multi-stage build.

FROM texlive/texlive:latest@sha256:a7ae4dfa9d521b5db14446872fa488b839021d1f604d0a3c74461784895f2a67

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    && rm -rf /var/lib/apt/lists/*

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/superfolha /app/server

WORKDIR /app

EXPOSE 8080

# Match render/railway: disk at /data, repos under {STATE_DIR}/repos/{uuid}
ENV STATE_DIR=/data
ENV PORT=8080

RUN mkdir -p /data

ENTRYPOINT ["/app/server"]
