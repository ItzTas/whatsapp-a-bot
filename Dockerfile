FROM golang:1.27.2-alpine3.24 AS build

# hadolint ignore=DL3018
RUN apk add --no-cache gcc musl-dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build \
      -trimpath \
      -tags sqlite_omit_load_extension \
      -ldflags '-s -w -linkmode external -extldflags "-static"' \
      -o /out/whatsapp-a-bot . \
    && mkdir -p /out/data

FROM alpine:3.24

# hadolint ignore=DL3018
RUN apk add --no-cache ffmpeg gallery-dl imagemagick imagemagick-webp yt-dlp \
    && addgroup -S -g 65532 nonroot \
    && adduser -S -D -H -u 65532 -G nonroot nonroot

COPY --from=build /out/whatsapp-a-bot /usr/local/bin/whatsapp-a-bot
COPY --from=build --chown=nonroot:nonroot /out/data /data

WORKDIR /data
VOLUME /data

USER nonroot:nonroot

HEALTHCHECK NONE

ENTRYPOINT ["/usr/local/bin/whatsapp-a-bot"]
