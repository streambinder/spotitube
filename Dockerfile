# docker run -it --rm -p 65535:65535/tcp -v ~/Music:/data -v ~/.cache:/cache ghcr.io/streambinder/spotitube
FROM golang:alpine AS builder
WORKDIR /workspace
COPY . .
RUN --mount=type=secret,id=SPOTIFY_ID \
    --mount=type=secret,id=SPOTIFY_KEY \
    --mount=type=secret,id=GENIUS_TOKEN \
    go mod download && \
    go build -ldflags="-s -w -X github.com/streambinder/spotitube/spotify.fallbackSpotifyID=$(cat /run/secrets/SPOTIFY_ID) -X github.com/streambinder/spotitube/spotify.fallbackSpotifyKey=$(cat /run/secrets/SPOTIFY_KEY) -X github.com/streambinder/spotitube/lyrics.fallbackGeniusToken=$(cat /run/secrets/GENIUS_TOKEN)"

FROM alpine:3
RUN apk add --no-cache ffmpeg yt-dlp && \
    mkdir /data && \
    mkdir /cache && \
    adduser -S spotitube && \
    chown -R spotitube /data /cache
USER spotitube
WORKDIR /data
ENV XDG_MUSIC_DIR=/data
ENV XDG_CACHE_HOME=/cache
COPY --from=builder /workspace/spotitube /usr/sbin/
HEALTHCHECK CMD [ "/usr/sbin/spotitube", "--help" ]
EXPOSE 65535/tcp
ENTRYPOINT ["/usr/sbin/spotitube"]
LABEL org.opencontainers.image.source=https://github.com/streambinder/spotitube
