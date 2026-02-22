# ==========================================
# Stage 1: Test Runner
# ==========================================
FROM golang:1.21-alpine AS test

RUN apk add --no-cache git gcc musl-dev make curl

WORKDIR /build

# Copy dependency files first (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Run unit tests
RUN go test -v -race -count=1 ./...

# ==========================================
# Stage 2: Builder
# ==========================================
FROM test AS builder

ARG VERSION=dev
ARG GIT_COMMIT=none

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-s -w \
    -X github.com/user/gocontainer/internal/cli.Version=${VERSION} \
    -X github.com/user/gocontainer/internal/cli.GitCommit=${GIT_COMMIT} \
    -X github.com/user/gocontainer/internal/cli.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /gocontainer ./cmd/gocontainer

# ==========================================
# Stage 3: Runtime Image
# ==========================================
FROM alpine:3.19 AS runtime

# Download Alpine minirootfs for use as container rootfs
RUN mkdir -p /rootfs && \
    wget -qO- https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.0-x86_64.tar.gz \
    | tar xz -C /rootfs

# Copy the binary
COPY --from=builder /gocontainer /usr/local/bin/gocontainer

# Metadata
LABEL maintainer="GoContainer Contributors"
LABEL description="Minimal container runtime using Linux namespaces and cgroups"
LABEL version="${VERSION}"

ENTRYPOINT ["gocontainer"]
CMD ["--help"]
