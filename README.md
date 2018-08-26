# SPOTITUBE

[![](https://goreportcard.com/badge/github.com/streambinder/spotitube)](https://goreportcard.com/report/github.com/streambinder/spotitube) [![](https://img.shields.io/github/downloads/streambinder/spotitube/total.svg)](http://www.somsubhra.com/github-release-stats/?username=streambinder&repository=spotitube)

Programmatically authenticate to your Spotify account and synchronize your music, keeping playlists files, metadata informations, album artworks, songs lyrics and maximizing _mp3_ files quality.

* * *

[![](https://raw.githubusercontent.com/streambinder/spotitube/master/assets/images/sample.gif)](#spotitube)

## What is

This project was born as per two needs:

1.  I wanted to learn some _GO-lang_ basics.
2.  I needed to automate the process of synchronize the songs I wanted to download. This process is composed by several phases:

	-   Keep track of music I want to download
	-   Find the best song file I can
	-   Download it
	-   Apply correct metadata

_spotitube_ basically solves these two major problems in a simple, elegant, but especially rapid way.

### How does it work

The solution I wrote to automate the process is covered by three major components:

1.  _Spotify_

	This component, once authenticated, is used to keep track of the music to synchronize (both via library or a playlist) and as database for the metadata to apply to every downloaded _mp3_.

2.  _YouTube_:

	This one is our free music shop, used to be queried to give us the best video it owns about the songs we're looking for. Once found, that one gets downloaded using a combination of `youtube-dl` and `ffmpeg` commands.

3.  Lyrics provider (_Genius_ or _lyrics.ovh_):

	You will go through this component if you'll enable automatic songs lyrics fetch: _Spotify_ informations about song will be used to find lyrics provided by two entities: _Genius_ and, eventually, if the first one doesn't own it, _lyrics.ovh_.

## What does it need

As already mentioned it heavily uses `youtube-dl` to download tracks from _YouTube_ and `ffmpeg` to convert them to _mp3_. You absolutely need them. Thus, it's written in `GO-lang`: assure you actually own it.

| Dependency   |        Version       | Dependency type |
| ------------ | :------------------: | :-------------: |
| `youtube-dl` | _none in particular_ |     Runtime     |
| `ffmpeg`     | _none in particular_ |     Runtime     |
| `golang`     |         1.7+         |   Compilation   |

## What about its reliability

[![](https://raw.githubusercontent.com/streambinder/spotitube/master/assets/images/sample_result.png)](#spotitube)

Several tests got made during the drawing up of the application and now I can say its pretty good at choosing the right song out of a list of keywords (such as the title and the user of any _YouTube_ video).

### Latest statistics

Latest verified statistics describes a sample of 396 songs, cumulative of different musical genres: _rock_, _pop_, _disco_ - _house_, _dubstep_ and _remixes_ -, _chamber music_, _soundtrack_, _folk_, _indie_, _punk_, and many others. Also, they belonged to several decades, with songs from 1975 or up to 2017. They were produced by many and very different artists, such as _Kodaline_, _Don Diablo_, _OneRepublic_, _The Cinematic Orchestra_, _Sigur Ros_, _Rooney_, _Royal Blood_, _Antonello Venditti_, _Skrillex_, _Savant_, _Knife Party_, _Yann Tiersen_, _Celine Dion_, _The Lumineers_, _alt-J_, _Mumford & Sons_, _Patrick Park_, _Jake Bugg_, _About Wayne_, _Arctic Monkeys_, _The Offspring_, _Maitre Gims_, _Thegiornalisti_, _Glee_ cast, _One Direction_, _Baustelle_, _Kaleo_, _La La Land_ cast, and many, many more.

The result of `spotitube` execution:

| Type               | Quantity (of 396) |
| ------------------ | :---------------: |
| Songs _not found_  |       **13**      |
| Found, but _wrong_ |       **22**      |
| Found, and _right_ |      **361**      |

In other words, we could say `spotitube` behaved as it was expected to both for _songs not found_ and _found, and right_. In fact, in the first case, the greatest part of the _not found_ songs were actually really not found on _YouTube_.

| Type    | Percentage |
| ------- | :--------: |
| Success |   **95%**  |
| Failure |   **5%**   |

**PS** The code can surely be taught to behave always better, but there will always be a small percentage of failures, caused by the _YouTube_ users/uploaders, which are unable to specify what a video actually is containing and synthesize it in a title that is not ambiguous (I'm thinking about, for example, the case of a really talented teenager who posts his first cover video, without specifying that it actually is a cover). The more you'll get involved on improve `spotitube`, the more you'll notice how lot of things are ambigous and thinking of a way to workaround this ambiguity would bring the project to be too much selective, losing useful results.

### How to install

#### Download package

| Platform                   | File                                                                           |
| -------------------------- | ------------------------------------------------------------------------------ |
| Debian-based distributions | [`spotitube.deb`](https://github.com/streambinder/spotitube/releases/latest)   |
| RedHat-based distributions | [`spotitube.rpm`](https://github.com/streambinder/spotitube/releases/latest)   |
| Solus-Project              | [`spotitube.eopkg`](https://github.com/streambinder/spotitube/releases/latest) |
| Generic binary             | [`spotitube.bin`](https://github.com/streambinder/spotitube/releases/latest)   |

#### Build it yourself

The way to build it is pretty straightforward:

```bash
git clone https://github.com/streambinder/spotitube
cd spotitube
# the following SPOTIFY_ID and SPOTIFY_KEY are bogus
# read the "Spotify application keys" section for further informations
SPOTIFY_ID=YJ5U6TSB317572L40EMQQPVEI2HICXFL SPOTIFY_KEY=4SW2W3ICZ3DPY6NWC88UFJDBCZJAQA8J make
# to install system-wide
sudo make install
# otherwise you'll find the binary inside ./bin
```

##### Spotify application keys

Behind this tool there's a Spotify application that gets called during the authentication phase, which Spotify gives permissions to, to read many informations, such as the user name, library and playlists.
When you use _Spotitube_ for the very first time, Spotify will ask you if you really want to grant this informations to it.

The Spotify application gets linked to this go-lang code using the `SPOTIFY_ID` and the `SPOITIFY_KEY` provided by Spotify to the user who created the application (me).
It's not a good deal to hardcode these application credentials into the source code (as I previously was doing), letting anyone to see and use the same ones and letting anyone to pretend to be _Spotitube_, being then able to steal such informations, hiding his real identity.
This is the reason behind the choice to hide those credentials from the source code, and applying - expliciting as environment variables - them during the compilation phase.
On the other hand, this unfortunately means that no one can compile the tool but me (or anyone else which the keys are granted to): if you want, you can easily create an application to the Spotify [developer area](https://beta.developer.spotify.com/dashboard/applications) and use your own credentials.

For the ones moving this way, keep in mind:

1.  `SPOTIFY_KEY` is associated to `SPOTIFY_ID`: if you create your own app, remember to override both values provided to you by Spotify developers dashboard;
2.  If you do not want to manually alter the code to make SpotiTube listen on different URI for Spotify authentication flow completion, you'd better set up `http://localhost:8080/callback` as callback URI of your Spotify app.

### How to use

```bash
# to download your music library
spotitube -folder ~/Music
# to download a specific $USERNAME accessible playlist via its URI
# look below for more informations on how to get that URI
spotitube -folder ~/Music -playlist spotify:user:$USERNAME:playlist:$PLAYLIST_ID
```

#### How to pull out URI from playlist

As some of users struggled to manage to have correct URI from a Spotify playlist, here's how you can get it:

[![](https://raw.githubusercontent.com/streambinder/spotitube/master/assets/images/sample_playlist.png)](#)

##### Empty or not recognized playlists on my Android phone

Android delegates the indexing of every media file stored into internal/external storage to a service called MediaScanner, which gets executed to find any new or deprecated file and to update a database filled with all those entries, MediaStore. This is basically done to let every app be faster to find files on storage, relying on this service rather than on specific implementations.

In few cases, some issues got encountered while testing SpotiTube generated playlists, recognized as empty on Android. After some investigations, it seems that MediaScanner defers the `.m3u` (playlist file) - the same with `.pls` file - effective parsing, in a actually not understood way: it immediately find the physical playlist file, but its informations will never be parsed.

A simple workaround for the ones experiencing this kind of issues, is to run this shell snippet:

```bash
adb shell "am broadcast -a android.intent.action.MEDIA_SCANNER_SCAN_FILE \
	-d file:///sdcard/path/to/playlist/file"
sleep 5
adb reboot
```

#### Additional flags

You may want to use some of the following input flags:

1.  `-fix <filename>`: try to find a better result for `<filename>`, which is an already downloaded (via SpotiTube) song
2.  `-disable-normalization`: disable songs volume normalization. Although volume normalization is really useful, as lot of songs gets downloaded with several `max_volume` values, resulting into some of them with very low volume level, this option (enabled by default) make the process slow down.
3.  `-disable-playlist-file`: disable automatic creation of playlist file, used to keep track of playlists songs.
4.  `-pls-file`: swap playlist file format, from `.m3u` - which is the default - to `.pls`.
5.  `-disable-lyrics`: disable download of songs lyrics and their application into `mp3`.
6.  `-disable-timestamp-flush`: disable automatic songs files timestamps flush to keep library/playlist order.
7.  `-disable-update-check`: disable automatic update check at startup.
8.  `-interactive`: enable interactive mode. This allows to eventually override `spotitube` decisions about which _YouTube_ result to pick, prompting for user input on every - legal - song it encounters.
9.  `-flush-metadata`: enable metadata informations flush also for songs that have been already synchronized.
10. `-flush-missing`: if `-flush-metadata` toggled, it will just populate empty id3 frames, instead of flushing any of those.
11. `-replace-local`: replace already downloaded (via `spotitube`) songs, if better ones get encountered.
12. `-clean-junks`: forcely batch remove temporary files that kept existing for any unattended runtime error.
13. `-version`: just print installed version.

#### Developers

For developers, maybe two additional flags could be really useful to simplify the troubleshooting and bugfixing process:

1.  `-log` will append every output line of the application to a logfile, located inside the `-folder` the music is getting synchronized in.
2.  `-debug` will show additional and detailed messages about the flow that brought the code to choose a song, instead of another, for example. Also, this flag will disable parallelism, in order to have a clearer and more ordered output.
3.  `-simulate` will make the process flow till the download step, without proceeding on its way. It's useful to check if searching for results and filtering steps are doing their job.
4.  `-disable-gui` will make the application output flow as a simple CLI application, dropping all the noise brought by the GUI-like aesthetics.

##### Footnote

The mobile application the sample result song is getting viewed and listened is the opensource [Phonograph](https://github.com/kabouzeid/phonograph), by [kabouzeid](https://github.com/kabouzeid).
