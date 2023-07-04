FROM golang:alpine as builder
WORKDIR /workspace
COPY go.mod .
COPY go.sum .
RUN go mod download
# git is garble runtime dependency
# https://github.com/bluekeyes/go-gitdiff/issues/30
RUN apk add --no-cache git
RUN go install mvdan.cc/garble@latest
COPY . .
RUN sed -iE "s/(fallbackSpotifyID += +)\"\"$/\1\"$SPOTIFY_ID\"/g" spotify/auth.go
RUN sed -iE "s/(fallbackSpotifyKey += +)\"\"$/\1\"$SPOTIFY_KEY\"/g" spotify/auth.go
RUN sed -iE "s/(fallbackGeniusToken += +)\"\"$/\1\"$GENIUS_TOKEN\"/g" lyrics/genius.go
RUN garble -literals -tiny -seed=random build

FROM alpine:latest
RUN apk add --no-cache ffmpeg yt-dlp
RUN mkdir /data
WORKDIR /data
ENV XDG_MUSIC_DIR=/data
COPY --from=builder /workspace/spotitube /usr/sbin/
ENTRYPOINT ["/usr/sbin/spotitube"]
LABEL org.opencontainers.image.source=https://github.com/streambinder/spotitube
