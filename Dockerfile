# docker run -it --rm -p 65535:65535/tcp -v ~/Music:/data -v ~/.cache:/cache ghcr.io/streambinder/spotitube
FROM golang:alpine AS builder
WORKDIR /workspace
COPY . .
RUN --mount=type=secret,id=SPOTIFY_ID \
    --mount=type=secret,id=SPOTIFY_KEY \
    --mount=type=secret,id=GENIUS_TOKEN \
    SPOTIFY_ID=$(tr -d '\n' </run/secrets/SPOTIFY_ID) && \
    SPOTIFY_KEY=$(tr -d '\n' </run/secrets/SPOTIFY_KEY) && \
    GENIUS_TOKEN=$(tr -d '\n' </run/secrets/GENIUS_TOKEN) && \
    go mod download && \
    go build -o spotitube -ldflags="-s -w -X \"github.com/streambinder/spotitube/spotify.fallbackSpotifyID=${SPOTIFY_ID}\" -X \"github.com/streambinder/spotitube/spotify.fallbackSpotifyKey=${SPOTIFY_KEY}\" -X \"github.com/streambinder/spotitube/lyrics.fallbackGeniusToken=${GENIUS_TOKEN}\"" .

FROM alpine:3
RUN apk add --no-cache ffmpeg yt-dlp && \
    mkdir /data && \
    mkdir /cache && \
    adduser -S spotitube && \
    chown -R spotitube /data /cache && \
    chmod -R 777 /cache
USER spotitube
WORKDIR /data
ENV XDG_MUSIC_DIR=/data
ENV XDG_CACHE_HOME=/cache
COPY --from=builder /workspace/spotitube /usr/sbin/
HEALTHCHECK CMD [ "/usr/sbin/spotitube", "--help" ]
EXPOSE 65535/tcp
ENTRYPOINT ["/usr/sbin/spotitube"]
LABEL org.opencontainers.image.source=https://github.com/streambinder/spotitube
