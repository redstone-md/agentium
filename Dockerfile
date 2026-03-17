FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/agentium ./cmd/agentium

# Ubuntu 24.04 moves chromium-browser to a transitional package that installs the snap,
# which is a poor fit for minimal container runtime images. Debian bookworm provides
# a regular apt-installable chromium package, so the runtime stays headful/Xvfb-friendly.
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    chromium \
    xvfb \
    ca-certificates \
    fonts-liberation \
    fonts-noto-color-emoji \
    libasound2 \
    libgbm1 \
    libgtk-3-0 \
    libnss3 \
    libx11-xcb1 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    xdg-utils \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/agentium /usr/local/bin/agentium
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV AGENTIUM_CHROME_BIN=/usr/bin/chromium
ENV DISPLAY=:99
EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]
