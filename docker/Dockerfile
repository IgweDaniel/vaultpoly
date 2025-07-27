# --- Build Stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy your source code into the build container
COPY . /build

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
# Build the plugin binary
RUN go build -v -o vaultpoly ./cmd/main.go

# Calculate SHA256 checksum
RUN sha256sum /build/vaultpoly > /build/SHA256SUMS

# --- Vault Stage ---
FROM hashicorp/vault:1.20
RUN apk add --no-cache curl jq
USER root

# Copy plugin and checksum from builder
COPY --from=builder /build/vaultpoly /vault/plugins/vaultpoly
COPY --from=builder /build/SHA256SUMS /vault/plugins/SHA256SUMS

COPY ./docker/config.hcl /vault/config/config.hcl
# Copy test script

RUN setcap cap_ipc_lock=+ep /vault/plugins/vaultpoly

# COPY ./docker/test.sh /test.sh
# RUN chmod 777 /test.sh
# RUN chmod +x /test.sh 

# Expose Vault port
EXPOSE 8200

