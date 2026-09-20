# Build stage: cross-compiles for the target platform from the build host.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
      -ldflags="-s -w -X github.com/coolapso/agent-skills-validator/cmd.Version=${VERSION}" \
      -o /out/agent-skills-validator .

# Runtime stage: the validator is a static binary that never touches the
# network, so an empty base image is enough.
FROM scratch

LABEL org.opencontainers.image.title="agent-skills-validator" \
      org.opencontainers.image.description="Validate Agent Skills against the public specification" \
      org.opencontainers.image.source="https://github.com/coolapso/agent-skills-validator" \
      org.opencontainers.image.licenses="MIT"

COPY --from=builder /out/agent-skills-validator /usr/bin/agent-skills-validator

# Mount the skills you want to validate on /data:
#   docker run --rm -v "$PWD:/data" ghcr.io/coolapso/agent-skills-validator validate skills/my-skill
WORKDIR /data
USER 65532:65532

ENTRYPOINT ["/usr/bin/agent-skills-validator"]
CMD ["--help"]
