# syntax=docker/dockerfile:1
ARG GO_VERSION=1.25-bookworm
ARG PLAYWRIGHT_VERSION=1.61.0
FROM golang:${GO_VERSION} AS service
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG APP_VERSION=dev
ARG BUILD_TIME=unknown

COPY service/ /svc/
WORKDIR /svc
RUN CGO_ENABLED=0 \
  go build -trimpath \
  -ldflags "-s -w -X main.version=${APP_VERSION} -X main.commit=${GIT_COMMIT} -X main.branch=${GIT_BRANCH} -X main.buildTime=${BUILD_TIME}" \
  -o dossier-service \
  .

FROM mcr.microsoft.com/playwright:v${PLAYWRIGHT_VERSION}-noble
LABEL org.opencontainers.image.title="docker-chromium-screenshot" \
      org.opencontainers.image.description="Render Markdown/JSON/jsonnet cards into infographic PNGs with Astro, D3, Playwright, Chromium and a Go metadata service" \
      org.opencontainers.image.source="https://github.com/s0cks/docker-chromium-screenshot" \
      org.opencontainers.image.licenses="MIT"

# CARDS_DIR is the one mount a card needs: cards/<id>/ carries its own
# card.md or layout.*, its own logo.*, theme.css and meta.* — see
# docs/build-a-card.md. cards/_shared/ (optional) holds anything shared
# across cards in the same mount.
ENV ASTRO_TELEMETRY_DISABLED=1 \
    CARDS_DIR=/app/cards \
    OUT_DIR=/app/out \
    SERVICE_BIN=/app/service/dossier-service

WORKDIR /app

COPY package.json ./
RUN npm i --no-audit --no-fund

COPY --from=service /svc/dossier-service /app/service/dossier-service
COPY astro.config.mjs ./
COPY scripts ./scripts
COPY src ./src
COPY cards ./cards

ENTRYPOINT ["node", "scripts/render.mjs" ]
CMD []
