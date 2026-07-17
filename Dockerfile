# syntax=docker/dockerfile:1

# --- frontend ---------------------------------------------------------------
FROM node:20-bookworm-slim AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# --- go backend ---------------------------------------------------------------
FROM golang:1.26-bookworm AS backend
WORKDIR /src
ENV GOPRIVATE=github.com/kushiemoon-dev/* GONOSUMDB=github.com/kushiemoon-dev/*
COPY go.mod go.sum ./
# BuildKit secret: never written to an image layer, only readable inside this RUN.
RUN --mount=type=secret,id=core_access_token \
    if [ -s /run/secrets/core_access_token ]; then \
      git config --global url."https://x-access-token:$(cat /run/secrets/core_access_token)@github.com/".insteadOf "https://github.com/"; \
    fi \
    && go mod download \
    && rm -f /root/.gitconfig
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o /out/flacidal-server ./cmd/server

# --- sldl (Soulseek client binary, pinned to the same release the desktop app installs) ---
FROM debian:bookworm-slim AS sldl
ARG TARGETARCH
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*
# Same pinned tag + per-arch asset names as sldlDownloadURLs in flacidal-core/sldl_installer.go.
RUN case "$TARGETARCH" in \
      amd64) SLDL_URL='https://github.com/fiso64/sockseek/releases/download/v3.0.4/sockseek_3.0.4_linux-x64.tar.gz' ;; \
      arm64) SLDL_URL='https://github.com/fiso64/sockseek/releases/download/v3.0.4/sockseek_3.0.4_linux-arm.tar.gz' ;; \
      *) echo "unsupported arch: $TARGETARCH" >&2; exit 1 ;; \
    esac \
    && curl -fsSL "$SLDL_URL" -o /tmp/sldl.tar.gz \
    && mkdir -p /out \
    && tar -xzf /tmp/sldl.tar.gz -C /out sockseek \
    && mv /out/sockseek /out/sldl \
    && chmod +x /out/sldl

# --- runtime ---------------------------------------------------------------
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates ffmpeg \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 1000 -s /usr/sbin/nologin flacidal

USER flacidal
WORKDIR /app
ENV HOME=/home/flacidal \
    PORT=8080 \
    FRONTEND_DIST_DIR=/app/frontend/dist

COPY --from=backend --chown=flacidal:flacidal /out/flacidal-server /app/flacidal-server
COPY --from=frontend --chown=flacidal:flacidal /src/frontend/dist /app/frontend/dist
COPY --from=sldl --chown=flacidal:flacidal /out/sldl /home/flacidal/.local/share/flacidal/sldl

EXPOSE 8080
VOLUME ["/home/flacidal/.flacidal", "/home/flacidal/Music"]
ENTRYPOINT ["/app/flacidal-server"]
