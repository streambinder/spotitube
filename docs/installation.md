# Installation

## Official releases

Binaries released officially include all the needed tokens and keys to make Spotitube work at its best (e.g. Spotify app ID and key, or Genius token).

To install, head to Spotitube [Releases](https://github.com/streambinder/spotitube/releases) page (binaries published for `{linux,darwin,windows}` × `{amd64,arm64}`), or pull via Docker:

```bash
docker pull ghcr.io/streambinder/spotitube:latest
```

> **Heads up:** the published Docker image is built for `linux/arm64` only. On `amd64` or other architectures, build the image locally from the `Dockerfile` shipped at the repository root, or grab the matching native binary from the Releases page.

## Custom build

Spotitube's been written to be as much vanilla Go as possible, so all the traditional Go build/install methods are supported:

```bash
go install github.com/streambinder/spotitube@latest
```

Or:

```bash
git clone https://github.com/streambinder/spotitube.git
cd spotitube
go build
go install
```

### Runtime prerequisites

Outside of Docker, Spotitube shells out to a couple of binaries and expects them on `PATH`:

- `ffmpeg` — used by the Processor to normalize volume and re-mux downloaded audio.
- `yt-dlp` — used by the YouTube provider to fetch the chosen result.

Install them via your package manager (e.g. `apt install ffmpeg yt-dlp`, `brew install ffmpeg yt-dlp`, `dnf install ffmpeg yt-dlp`). The published Docker image bundles both already.

### Embedding API keys

By default, Spotitube will use `SPOTIFY_ID`, `SPOTIFY_KEY` and `GENIUS_TOKEN` environment variables to authenticate to the corresponding APIs.
If those are not found, though, it will fall back to the fallback fields defined in the corresponding source code modules (which, in turn, are empty, by default).
In order to build a binary which contains these fields, the following formula can be used:

```bash
go build -ldflags="
    -X github.com/streambinder/spotitube/spotify.fallbackSpotifyID='awesomeSpotifyID'
    -X github.com/streambinder/spotitube/spotify.fallbackSpotifyKey='awesomeSpotifyKey'
    -X github.com/streambinder/spotitube/lyrics.fallbackGeniusToken='awesomeGeniusToken'
"
```
