# docker run -it --rm -p 65535:65535/tcp -v ~/Music:/data -v ~/.cache:/cache ghcr.io/streambinder/pango
FROM golang:alpine AS builder
WORKDIR /workspace
COPY . .
RUN --mount=type=secret,id=SPOTIFY_ID \
    --mount=type=secret,id=SPOTIFY_KEY \
    --mount=type=secret,id=GENIUS_TOKEN \
    SPOTIFY_ID="$(cat /run/secrets/SPOTIFY_ID 2>/dev/null || true)" && \
    SPOTIFY_KEY="$(cat /run/secrets/SPOTIFY_KEY 2>/dev/null || true)" && \
    GENIUS_TOKEN="$(cat /run/secrets/GENIUS_TOKEN 2>/dev/null || true)" && \
    go mod download && \
    go build -o pango -ldflags="-s -w -X \"github.com/streambinder/pango/spotify.fallbackSpotifyID=${SPOTIFY_ID}\" -X \"github.com/streambinder/pango/spotify.fallbackSpotifyKey=${SPOTIFY_KEY}\" -X \"github.com/streambinder/pango/lyrics.fallbackGeniusToken=${GENIUS_TOKEN}\"" .

FROM alpine:3
RUN apk add --no-cache ffmpeg yt-dlp && \
    mkdir /data && \
    mkdir /cache && \
    adduser -S pango && \
    chown -R pango /data /cache && \
    chmod -R 777 /cache
USER pango
WORKDIR /data
ENV XDG_MUSIC_DIR=/data
ENV XDG_CACHE_HOME=/cache
COPY --from=builder /workspace/pango /usr/sbin/
HEALTHCHECK CMD [ "/usr/sbin/pango", "--help" ]
EXPOSE 65535/tcp
ENTRYPOINT ["/usr/sbin/pango"]
LABEL org.opencontainers.image.source=https://github.com/streambinder/pango
