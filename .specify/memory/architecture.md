# Pango Architecture Baseline

**Status**: Baseline. Recorded on 2026-10-26 from the design
documents under `docs/` and from the code on the master branch. A
change to this baseline needs a decision in the register under
`.specify/memory/decisions/`.

## Names

- **Pango** — the application. The project carried the name
  Spotitube until the rename of 2026-10-13. The personal playlist
  of the owner keeps the legacy name `spotitube-sync`.
- A **track** is one song, identified by its Spotify identifier.
- A **provider** supplies a downloadable asset for a track.
- A **composer** supplies a lyrics text for a track.

## Principles

- Spotify metadata defines track identity. Providers and composers
  enrich a track; neither redefines it.
- Every source is queried and the best result wins. One failing
  source never aborts the others.
- An enrichment failure degrades. A track with audio and artwork
  is never discarded over missing lyrics.
- The ID3 frames in the installed file are the library index. A
  later run learns the library state from the files themselves.
- Credentials stay out of source code. A credential comes from an
  environment variable or from a release-time fallback.

## Pipeline

`sync` runs an assembly line of routines. Each routine owns one
stage, consumes a queue, and passes track objects onward.

- **Index**: scans the output folder, reads the Spotify identifier
  frame of every installed file, and builds the library index. The
  index detects filename collisions and already installed tracks.
- **Auth**: authenticates to Spotify and signals the fetcher. The
  OAuth callback listens on the loopback port 65535. The session
  token persists in the cache directory as `session.json`.
- **Fetch**: reads the requested collections (library, playlists,
  albums, tracks, fixes) from the Spotify API and emits one track
  object per track. A fix request reads the Spotify identifier of
  a local file, removes the file, and re-queues the track.
- **Decide**: queries every asset provider for each track, scores
  the matches, and keeps the best upstream URL. Manual mode
  replaces the choice with a URL that the user supplies.
- **Collect**: downloads the asset, the lyrics, and the artwork of
  a track, concurrently per track and across tracks.
- **Process**: normalizes loudness to -14 LUFS through `ffmpeg`
  and writes every collected field as ID3 frames.
- **Install**: moves the processed file to its final path,
  `Artist - Title.mp3` under the output folder.
- **Mix**: writes one playlist file per synchronized playlist, in
  M3U or PLS encoding, with the tracks that the run installed or
  that the index already held.

The `audit` command runs the index, auth, and fetch stages only
and reports every filename collision in one pass. The command
writes nothing. The `attach`, `lookup`, `show`, and `reset`
commands read or write single files or the stored session.

## Data Model

`entity.Track` carries one track through the pipeline: identifier,
title, artists, album, artwork URL and data, duration, lyrics
text, track number, year, and upstream URL. The title that the
lyrics composers and the providers search is the song title, with
variant suffixes stripped.

The encoder writes the track as ID3v2 frames: standard text frames
for title, artist, album, year, and track number; an attached
picture for the artwork; an unsynchronised lyrics frame for the
lyrics; and user-defined text frames for the Spotify identifier,
the artwork URL, the duration, and the upstream URL.

## Sources

- **Spotify** (package `spotify`): collections, track metadata,
  and authentication. Credentials: `SPOTIFY_ID` and `SPOTIFY_KEY`
  environment variables, or release-time fallbacks.
- **YouTube** (package `provider`): asset search and match
  scoring.
- **Genius** (package `lyrics`): search through the Genius API and
  lyrics extraction from the result page. The credential is the
  `GENIUS_TOKEN` environment variable or the release-time
  fallback.
- **LRCLIB** (package `lyrics`): exact-match lyrics lookup through
  the LRCLIB API, with plain and synchronized texts.

Lyrics collection caches the winning text in the cache directory,
in one file per track, named from the track identifier. A cache
hit answers a later lookup without a network call.

## Local State

The cache directory follows the XDG cache convention. The cache
holds the session token, the downloaded asset of a track in
flight, the artwork of a track in flight, and one lyrics file per
track. The output folder follows the XDG music convention and
holds the installed library and the playlist files.

## Library Consumers

External tools in the `unciae` repository import the packages of
this module and pin the module version: `inflarics` imports
`entity`, `lyrics`, and `sys` to fetch lyrics and write them into
local files; `pluffer` imports `spotify` for authentication and
collection fetching. A release tag publishes the module version
that the consumers pin.

## Deployment

A container image publishes to GHCR on every master push. A
production host runs the image under a daily systemd timer through
Imperium, against the real library of the owner. Release tags
publish binaries for macOS, Linux, and Windows, in amd64 and arm64
variants, with the fallback credentials injected at link time.
