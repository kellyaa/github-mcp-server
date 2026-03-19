FROM ghcr.io/github/github-mcp-server AS upstream

FROM golang:1.25.8-alpine@sha256:8e02eb337d9e0ea459e041f1ee5eece41cbb61f1d83e7d883a3e2fb4862063fa AS build
ARG VERSION="dev"

WORKDIR /build

RUN --mount=type=cache,target=/var/cache/apk \
    apk add git

# Copy source code
COPY . .

# Build the server with our modified token files
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /bin/github-mcp-server ./cmd/github-mcp-server

# Use the upstream image as base, replace only the binary
FROM upstream
COPY --from=build /bin/github-mcp-server /server/github-mcp-server
