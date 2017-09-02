# SPOTIFY-DL

## What is

_spotify-dl_'s birth is caused by two major needs of mine:

1. Was looking for something exciting and fun to work on in order to learn some of _GO lang_ basics.
2. Needed some way to speed up the process of downloading my music and automate the process of:

    1. Keep track of music I want to download
    2. Find the best song file I can (both from _YouTube_ or other sources)
    3. Download it
    4. Apply correct metadata

Thus, my music was recently moving from an offline library kept on my desktop computer to a cloud one, on _Spotify_. And I didn't like that.

_spotify-dl_ basically solves these two major problems in a simple and elegant way (at least for me).

## How does it work

The whole project is actually a wrapper of two main components:

1. _Spotify APIs_: this part is delegated to authenticate the user and guide the application to his library (or playlist);
2. _YouTube_: this part wraps the amazing `youtube-dl` tool, providing a smart way to find the best result on _YouTube_ for the _Spotify_ track to download.

So, the flow results basically to be:

1. _Spotify_ authentication via its _API_;
2. Pull down private user library (or - if specifically asked via command-line - a playlist) and fill a list with these data;
3. Filter this list with the offline songs (for this, it checks the content of the `~/Music` folder, of the one passed via `-folder` flag);
4. Search on _YouTube_ for any song kept in this _delta_ list;
5. Download from _YouTube_ (with best audio quality and convert it using `ffmpeg` to _MP3_);
6. Apply _Spotify_ provided metadata to the downloaded song;
7. Move the song into `~/Music` (or the one passed via `-folder` flag) path.

## What does it need

As already mentioned it heavily uses `youtube-dl` to download tracks from _YouTube_ and `ffmpeg` to convert them to _MP3_. You absolutely need them. Thus, it's written in `GO lang`: assure you actually own it.

Resuming:

1. `youtube-dl`
2. `ffmpeg`
3. `golang` (`1.7` or major)

## How to install

The way to install it is pretty straightforward:

```bash
git clone https://github.com/streambinder/spotify-dl
cd spotify-dl
make
sudo make install

# to download your music library
spotify-dl -folder ~/Music
# to download a specific - public - playlist
spotify-dl -folder ~/Music -playlist spotify:user:spotify:playlist:37i9dQZF1DWSQScAbo5nGF
```
