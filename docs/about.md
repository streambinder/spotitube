# About

![demo](assets/demo.gif)

Spotitube is a CLI application to authenticate to Spotify account, fetch music collections — such as account library, playlists, albums or specific tracks —, look them up on a defined set of providers (currently YouTube and Qobuz), download them and inflate the downloaded assets with metadata collected from Spotify.

Downloaded tracks are further enriched with lyrics fetched from Genius and LRCLIB, including synced LRC when available.

## Usage

By default, Spotitube will synchronize user's library:

```bash
spotitube sync
```

In order to synchronize further set of tracks, use the dedicated flags:

```bash
spotitube sync --playlist spotitube-sync \
    --album https://open.spotify.com/album/6Jx4cGhWHewTcfKDJKguBQ?si=426ac1fd0fbe4cab \
    --playlist-tracks 5s9gvhZDtfTaM8VMfBtssy \
    --track 6SdAztAqklk1zAmUHhU4N7 \
    --fix /path/to/already/downloaded/track.mp3
```

As showed in the previous example, there are several ways to indicate the ID of a resource — be it a playlist, album or track:
regardless Spotitube is given a full URL to that resource (e.g. `https://open.spotify.com/playlist/2wyZKlaKzPEUurb6KshAwQ?si=426ac1fd0fbe4cab`), a URI (e.g. `spotify:playlist:2wyZKlaKzPEUurb6KshAwQ`) or an ID (e.g. `2wyZKlaKzPEUurb6KshAwQ`), it should be smart enough to solve the effective ID resolution all by itself.

Furthermore, in case of playlist, automatic aliasing of personal playlist names into their ID is applied: this enables passing playlist by name instead of ID in case user wants to synchronize personal playlists.

By default, Spotitube uses XDG Base/User Directory Specification to resolve user's Music folder (which usually maps to `~/Music`), but it can be obviously overridden using a dedicated flag:

```bash
spotitube sync -o ~/MyMusic
```

Additional `sync` flags worth knowing:

- `--library` / `-l` — explicitly synchronize the library (auto-enabled if no collection flag is passed).
- `--library-limit N` — cap the number of library tracks fetched (`0` = unlimited, default).
- `--playlist-encoding {m3u,pls}` — playlist file format produced by the Mixer (default `m3u`).
- `--plain` — disable the fancy TUI; emit plain line-oriented output (useful for cron/CI).
- `--manual` / `-m` — prompt for a user-supplied provider URL per track instead of letting the Decider pick.

### Subcommands

Beyond `sync`, the following subcommands are available — list them via `spotitube --help`:

- `auth` — establish a Spotify session and persist the OAuth token to `${XDG_CACHE_HOME:-~/.cache}/spotitube/session.json`. Pass `--logout` / `-l` to wipe the cached token before re-authenticating.
- `attach` — attach Spotify metadata (including the Spotify ID embedded in a custom ID3 frame) to an existing local file.
- `lookup` — query Spotify for a resource and print its metadata without downloading.
- `show` — show the Spotify metadata embedded in a local file.
- `reset` — remove the cached session and any local state.

### Authentication scopes

`spotitube auth` requests both read and write OAuth scopes on the user's account:

- `user-library-read`, `user-library-modify`
- `playlist-read-private`, `playlist-read-collaborative`
- `playlist-modify-public`, `playlist-modify-private`

The modify scopes are requested even though `sync` is read-only against Spotify — they reserve the ability to push library/playlist changes from local state in the future. Approve them only if you trust your Spotify app's client ID and secret.

### Docker

In order to make Spotitube work via Docker, it has to expose its dedicated port (i.e. 65535) and mount both the cache and the music directories as volumes:

```bash
docker run -it --rm \
    -p 65535:65535/tcp \
    -v ~/.cache:/cache \
    -v ~/Music:/data \
    ghcr.io/streambinder/spotitube --help
```

### Headless

The only real friction running Spotitube headless is the OAuth redirect.
Spotify [requires](https://developer.spotify.com/documentation/web-api/concepts/redirect_uri) the callback to be either an HTTPS URL or one of the loopback literals `http://127.0.0.1:PORT` / `http://[::1]:PORT`.
Spotitube uses the loopback form (`http://127.0.0.1:65535/callback`), which means the browser completing the auth must be able to reach that callback on the host actually running Spotitube.

The simplest way to bridge a desktop browser to a headless server is an SSH local port forward — no DNS, no extra DNAT, no TLS termination:

```bash
ssh -L 65535:127.0.0.1:65535 user@server
# on the server, in the forwarded session:
spotitube auth
```

Then open the URL Spotitube prints in your local browser.
Spotify will redirect to `http://127.0.0.1:65535/callback`, the tunnel hands the request through to the server, and the session token is persisted server-side at `${XDG_CACHE_HOME:-~/.cache}/spotitube/session.json`.
After `auth` returns, the tunnel and SSH session can be closed; subsequent `spotitube` invocations on the server reuse the cached token.

### Manual mode

It might very well happen that Spotitube is either not able to find a track asset on given providers (e.g. YouTube) or that it chooses the wrong one.
In such cases, it is possible to manually choose and pass the right asset to Spotitube, using the `--manual` flag:

```bash
spotitube sync --manual --track 6SdAztAqklk1zAmUHh
```

Spotitube will patiently wait for the user to pass the URL of the track asset to download.
This can come in useful in cases where the track has been already downloaded wrong and user wants to touch on it:

```bash
spotitube sync --manual --fix /path/to/already/downloaded/track.mp3
```
