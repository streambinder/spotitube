# docker run -it --rm -p 65535:65535/tcp -v ~/Music:/data -v ~/.cache:/cache ghcr.io/streambinder/spotitubeFROM golang:alpine as builder
FROM golang:alpine as builder
WORKDIR /workspace
COPY . .
RUN go mod download
RUN --mount=type=secret,id=SPOTIFY_ID \
    --mount=type=secret,id=SPOTIFY_KEY \
    --mount=type=secret,id=GENIUS_TOKEN \
    go build -ldflags="-s -w -X github.com/streambinder/spotitube/spotify.fallbackSpotifyID=$(cat /run/secrets/SPOTIFY_ID) -X github.com/streambinder/spotitube/spotify.fallbackSpotifyKey=$(cat /run/secrets/SPOTIFY_KEY) -X github.com/streambinder/spotitube/lyrics.fallbackGeniusToken=$(cat /run/secrets/GENIUS_TOKEN)"

FROM alpine:latest
RUN apk add --no-cache ffmpeg yt-dlp
RUN mkdir /data
RUN mkdir /cache
WORKDIR /data
ENV XDG_MUSIC_DIR=/data
ENV XDG_CACHE_HOME=/cache
COPY --from=builder /workspace/spotitube /usr/sbin/
EXPOSE 65535/tcp
ENTRYPOINT ["/usr/sbin/spotitube"]
LABEL org.opencontainers.image.source=https://github.com/streambinder/spotitube
