# Installation

## Official releases

Binaries released officially include all the needed tokens and keys to make Spotitube work at its best (e.g. Spotify app ID and key, or Genius token).

To install, head to Spotitube [Releases](https://github.com/streambinder/spotitube/releases) page, or pull via Docker:

```bash
docker pull ghcr.io/streambinder/spotitube:latest
```

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
