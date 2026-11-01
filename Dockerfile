FROM golang:1.27.2-alpine3.24 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build \
      -trimpath \
      -ldflags '-s -w' \
      -o /out/whatsapp-a-bot . \
    && mkdir -p /out/data

FROM alpine:3.24

# hadolint ignore=DL3018
RUN apk add --no-cache ffmpeg gallery-dl imagemagick imagemagick-webp tesseract-ocr tesseract-ocr-data-eng tesseract-ocr-data-por yt-dlp \
    && addgroup -S -g 65532 nonroot \
    && adduser -S -D -H -u 65532 -G nonroot nonroot

COPY --from=build /out/whatsapp-a-bot /usr/local/bin/whatsapp-a-bot
COPY --from=build --chown=nonroot:nonroot /out/data /data

WORKDIR /data
VOLUME /data

USER nonroot:nonroot

HEALTHCHECK NONE

ENTRYPOINT ["/usr/local/bin/whatsapp-a-bot"]
