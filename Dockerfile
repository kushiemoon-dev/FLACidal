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
RUN VERSION=$(grep -m1 '"version"' wails.json | sed -E 's/.*"version": *"([^"]+)".*/\1/') \
    && CGO_ENABLED=1 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/flacidal-server ./cmd/server

# --- runtime ---------------------------------------------------------------
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates ffmpeg aubio-tools \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 1000 -s /usr/sbin/nologin flacidal

USER flacidal
WORKDIR /app
ENV HOME=/home/flacidal \
    PORT=8080 \
    LISTEN_HOST=0.0.0.0 \
    FRONTEND_DIST_DIR=/app/frontend/dist

COPY --from=backend --chown=flacidal:flacidal /out/flacidal-server /app/flacidal-server
COPY --from=frontend --chown=flacidal:flacidal /src/frontend/dist /app/frontend/dist

EXPOSE 8080
VOLUME ["/home/flacidal/.flacidal", "/home/flacidal/Music"]
ENTRYPOINT ["/app/flacidal-server"]
