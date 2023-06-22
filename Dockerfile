FROM golang:alpine as builder
WORKDIR /workspace
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -ldflags "-s -w"

FROM alpine:latest
RUN apk add --no-cache ffmpeg yt-dlp
RUN mkdir /data
WORKDIR /data
COPY --from=builder /workspace/spotitube /usr/sbin/
ENTRYPOINT ["/usr/sbin/spotitube"]
LABEL org.opencontainers.image.source=https://github.com/streambinder/spotitube
