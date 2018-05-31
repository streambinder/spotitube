package track

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	spttb_system "system"

	"github.com/bogem/id3v2"
	"github.com/kennygrant/sanitize"
	"github.com/zmb3/spotify"
)

// Has : return True if Tracks contains input Track
func (tracks Tracks) Has(track Track) bool {
	for _, havingTrack := range tracks {
		if strings.ToLower(havingTrack.Filename) == strings.ToLower(track.Filename) {
			return true
		}
	}
	return false
}

// CountOffline : return offline (local) songs count from Tracks
func (tracks Tracks) CountOffline() int {
	return len(tracks) - tracks.CountOnline()
}

// CountOnline : return online songs count from Tracks
func (tracks Tracks) CountOnline() int {
	var counter int
	for _, track := range tracks {
		if !track.Local {
			counter++
		}
	}
	return counter
}

// OpenLocalTrack : parse local filename track informations into a new Track object
func OpenLocalTrack(filename string) (Track, error) {
	if !spttb_system.FileExists(filename) {
		return Track{}, fmt.Errorf(fmt.Sprintf("%s does not exist", filename))
	}
	trackMp3, err := id3v2.Open(filename, id3v2.Options{Parse: true})
	if err != nil {
		return Track{}, fmt.Errorf(fmt.Sprintf("Cannot read id3 tags from \"%s\": %s", filename, err.Error()))
	}
	track := Track{
		Title:         TagGetFrame(trackMp3, ID3FrameTitle),
		Song:          TagGetFrame(trackMp3, ID3FrameSong),
		Artist:        TagGetFrame(trackMp3, ID3FrameArtist),
		Album:         TagGetFrame(trackMp3, ID3FrameAlbum),
		Year:          TagGetFrame(trackMp3, ID3FrameYear),
		Featurings:    strings.Split(TagGetFrame(trackMp3, ID3FrameTrackNumber), "|"),
		Genre:         TagGetFrame(trackMp3, ID3FrameGenre),
		TrackNumber:   0,
		TrackTotals:   0,
		Duration:      0,
		SongType:      parseType(TagGetFrame(trackMp3, ID3FrameTitle)),
		Image:         "",
		URL:           TagGetFrame(trackMp3, ID3FrameYouTubeURL),
		Filename:      "",
		FilenameTemp:  "",
		FilenameExt:   spttb_system.SongExtension,
		SearchPattern: "",
		Lyrics:        TagGetFrame(trackMp3, ID3FrameLyrics),
		Local:         true,
	}

	if trackNumber, trackNumberErr := strconv.Atoi(TagGetFrame(trackMp3, ID3FrameTrackNumber)); trackNumberErr == nil {
		track.TrackNumber = trackNumber
	}
	if trackTotals, trackTotalsErr := strconv.Atoi(TagGetFrame(trackMp3, ID3FrameTrackTotals)); trackTotalsErr == nil {
		track.TrackTotals = trackTotals
	}
	if duration, durationErr := strconv.Atoi(TagGetFrame(trackMp3, ID3FrameDuration)); durationErr == nil {
		track.Duration = duration
	}

	track.Filename, track.FilenameTemp = parseFilename(track)

	track.SearchPattern = strings.Replace(track.FilenameTemp[1:], "-", " ", -1)

	trackMp3.Close()
	return track, nil
}

// ParseSpotifyTrack : parse Spotify track into a new Track object
func ParseSpotifyTrack(spotifyTrack spotify.FullTrack, spotifyAlbum spotify.FullAlbum) Track {
	track := Track{
		Title:  spotifyTrack.SimpleTrack.Name,
		Artist: (spotifyTrack.SimpleTrack.Artists[0]).Name,
		Album:  spotifyTrack.Album.Name,
		Year: func() string {
			if spotifyAlbum.ReleaseDatePrecision == "year" {
				return spotifyAlbum.ReleaseDate
			} else if strings.Contains(spotifyAlbum.ReleaseDate, "-") {
				return strings.Split(spotifyAlbum.ReleaseDate, "-")[0]
			}
			return "0000"
		}(),
		Featurings: func() []string {
			var featurings []string
			if len(spotifyTrack.SimpleTrack.Artists) > 1 {
				for _, artistItem := range spotifyTrack.SimpleTrack.Artists[1:] {
					featurings = append(featurings, artistItem.Name)
				}
			}
			return featurings
		}(),
		Genre: func() string {
			if len(spotifyAlbum.Genres) > 0 {
				return spotifyAlbum.Genres[0]
			}
			return ""
		}(),
		TrackNumber:   spotifyTrack.SimpleTrack.TrackNumber,
		TrackTotals:   len(spotifyAlbum.Tracks.Tracks),
		Duration:      spotifyTrack.SimpleTrack.Duration / 1000,
		Image:         spotifyTrack.Album.Images[0].URL,
		URL:           "",
		Filename:      "",
		FilenameTemp:  "",
		FilenameExt:   spttb_system.SongExtension,
		SearchPattern: "",
		Lyrics:        "",
		Local:         false,
	}

	track.SongType = parseType(track.Title)
	track.Title, track.Song = parseTitle(track.Title, track.Featurings)

	track.Album = strings.Replace(track.Album, "[", "(", -1)
	track.Album = strings.Replace(track.Album, "]", ")", -1)
	track.Album = strings.Replace(track.Album, "{", "(", -1)
	track.Album = strings.Replace(track.Album, "}", ")", -1)

	track.Filename, track.FilenameTemp = parseFilename(track)

	track.SearchPattern = strings.Replace(track.FilenameTemp[1:], "-", " ", -1)

	if spttb_system.FileExists(track.FilenameFinal()) {
		track.Local = true
	}

	if track.Local {
		track.URL = track.GetID3Frame(ID3FrameYouTubeURL)
		track.Lyrics = track.GetID3Frame(ID3FrameLyrics)
	}

	return track
}

// FilenameFinal : return Track final filename
func (track Track) FilenameFinal() string {
	return track.Filename + track.FilenameExt
}

// FilenameTemporary : return Track temporary filename
func (track Track) FilenameTemporary() string {
	return track.FilenameTemp + track.FilenameExt
}

// FilenameArtwork : return Track artwork filename
func (track Track) FilenameArtwork() string {
	return "." + strings.Split(track.Image, "/")[len(strings.Split(track.Image, "/"))-1] + ".jpg"
}

// TempFiles : return strings array containing all possible junk file names
func (track Track) TempFiles() []string {
	var tempFiles []string
	for _, fnamePrefix := range []string{track.FilenameTemporary(), track.FilenameTemp, track.FilenameArtwork()} {
		tempFiles = append(tempFiles, fnamePrefix)
		for _, fnameJunkSuffix := range JunkSuffixes {
			tempFiles = append(tempFiles, fnamePrefix+fnameJunkSuffix)
		}
	}
	return tempFiles
}

// Seems : return nil error if sequence is input sequence string matches with Track
func (track Track) Seems(sequence string) error {
	if err := track.SeemsByWordMatch(sequence); err != nil {
		return err
	}
	if strings.Contains(strings.ToLower(sequence), "full album") {
		return errors.New("Item seems to be pointing to an album, not to a song")
	}
	for _, songType := range SongTypes {
		if SeemsType(sequence, songType) && track.SongType != songType {
			return errors.New("Songs seem to be of different types")
		}
	}
	return nil
}

// SeemsByWordMatch : return nil error if Track song name, artist and featurings are contained in sequence
func (track Track) SeemsByWordMatch(sequence string) error {
	sequence = sanitize.Name(strings.ToLower(sequence))
	for _, trackItem := range append([]string{track.Song, track.Artist}, track.Featurings...) {
		trackItem = strings.ToLower(trackItem)
		if len(trackItem) > 7 && trackItem[:7] == "cast of" {
			trackItem = strings.Replace(trackItem, "cast of", "", -1)
		} else if len(trackItem) > 5 && trackItem[len(trackItem)-5:] == " cast" {
			trackItem = strings.Replace(trackItem, "cast", "", -1)
		}
		trackItem = strings.Replace(trackItem, " & ", " and ", -1)
		if strings.Contains(trackItem, " and ") {
			trackItem = strings.Split(trackItem, " and ")[0]
		}
		trackItem = strings.TrimSpace(trackItem)
		trackItem = sanitize.Name(trackItem)
		if !strings.Contains(sequence, trackItem) {
			return errors.New("Songs seem to be mismatching by words comparison")
		}
	}
	return nil
}

// JunkWildcards : return strings array containing all possible junk filenames wilcards
func JunkWildcards() []string {
	var junkWildcards []string
	for _, junkSuffix := range JunkSuffixes {
		junkWildcards = append(junkWildcards, ".*"+junkSuffix)
	}
	return append(junkWildcards, ".*.mp3")
}

// TagGetFrame : get input frame from open input Tag
func TagGetFrame(tag *id3v2.Tag, frame int) string {
	switch frame {
	case ID3FrameTitle:
		return tag.Title()
	case ID3FrameSong:
		return TagGetFrameSong(tag)
	case ID3FrameArtist:
		return tag.Artist()
	case ID3FrameAlbum:
		return tag.Album()
	case ID3FrameGenre:
		return tag.Genre()
	case ID3FrameYear:
		return tag.Year()
	case ID3FrameFeaturings:
		return TagGetFrameFeaturings(tag)
	case ID3FrameTrackNumber:
		return TagGetFrameTrackNumber(tag)
	case ID3FrameTrackTotals:
		return TagGetFrameTrackTotals(tag)
	case ID3FrameArtwork:
		return TagGetFrameArtwork(tag)
	case ID3FrameLyrics:
		return TagGetFrameLyrics(tag)
	case ID3FrameYouTubeURL:
		return TagGetFrameYouTubeURL(tag)
	case ID3FrameDuration:
		return TagGetFrameDuration(tag)
	}
	return ""
}

// TagGetFrameSong : get track song title frame from input Tag
func TagGetFrameSong(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "song" {
				return comment.Text
			}
		}
	}
	return ""
}

// TagGetFrameSong : get track song title frame from input Tag
func TagGetFrameFeaturings(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "featurings" {
				return comment.Text
			}
		}
	}
	return ""
}

// TagGetFrameTrackNumber : get track number frame from input Tag
func TagGetFrameTrackNumber(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Track number/Position in set"))) > 0 {
		for _, frameText := range tag.GetFrames(tag.CommonID("Track number/Position in set")) {
			text, ok := frameText.(id3v2.TextFrame)
			if ok {
				return text.Text
			}
		}
	}
	return ""
}

// TagGetFrameYouTubeURL : get youtube URL frame from input Tag
func TagGetFrameTrackTotals(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "trackTotals" {
				return comment.Text
			}
		}
	}
	return ""
}

// TagGetFrameArtwork : get artwork frame from input Tag
func TagGetFrameArtwork(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Attached picture"))) > 0 {
		for _, framePicture := range tag.GetFrames(tag.CommonID("Attached picture")) {
			picture, ok := framePicture.(id3v2.PictureFrame)
			if ok {
				return string(picture.Picture)
			}
		}
	}
	return ""
}

// TagGetFrameLyrics : get lyrics frame from input Tag
func TagGetFrameLyrics(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))) > 0 {
		for _, frameLyrics := range tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription")) {
			lyrics, ok := frameLyrics.(id3v2.UnsynchronisedLyricsFrame)
			if ok {
				return lyrics.Lyrics
			}
		}
	}
	return ""
}

// TagGetFrameYouTubeURL : get youtube URL frame from input Tag
func TagGetFrameYouTubeURL(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "youtube" {
				return comment.Text
			}
		}
	}
	return ""
}

// TagGetFrameDuration : get duration frame from input Tag
func TagGetFrameDuration(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "duration" {
				return comment.Text
			}
		}
	}
	return ""
}

// TagHasFrame : return True if open input Tag has valued input frame
func TagHasFrame(tag *id3v2.Tag, frame int) bool {
	return TagGetFrame(tag, frame) != ""
}

// GetID3Frame : get Track ID3 input frame string value
func (track Track) GetID3Frame(frame int) string {
	tag, err := id3v2.Open(track.FilenameFinal(), id3v2.Options{Parse: true})
	if tag == nil || err != nil {
		return ""
	}
	defer tag.Close()
	return TagGetFrame(tag, frame)
}

// HasID3Frame : return True if Track has input ID3 frame
func (track *Track) HasID3Frame(frame int) bool {
	return track.GetID3Frame(frame) != ""
}

// SearchLyrics : search Track lyrics, eventually throwing returning error
func (track *Track) SearchLyrics() error {
	geniusLyrics, geniusErr := searchLyricsGenius(track)
	if geniusErr == nil {
		track.Lyrics = geniusLyrics
		return nil
	}
	ovhLyrics, ovhErr := searchLyricsOvh(track)
	if ovhErr == nil {
		track.Lyrics = ovhLyrics
		return nil
	}
	return ovhErr
}

// SeemsType : return True if input sequence matches with selected input songType variant
func SeemsType(sequence string, songType int) bool {
	var songTypeAliases []string
	if songType == SongTypeLive {
		songTypeAliases = []string{"@", "live", "perform", "tour"}
		for _, year := range spttb_system.MakeRange(1950, 2050) {
			songTypeAliases = append(songTypeAliases, []string{strconv.Itoa(year), "'" + strconv.Itoa(year)[2:]}...)
		}
	} else if songType == SongTypeCover {
		songTypeAliases = []string{"cover", "vs"}
	} else if songType == SongTypeRemix {
		songTypeAliases = []string{"remix", "radio edit"}
	} else if songType == SongTypeAcoustic {
		songTypeAliases = []string{"acoustic"}
	} else if songType == SongTypeKaraoke {
		songTypeAliases = []string{"karaoke", "instrumental"}
	} else if songType == SongTypeParody {
		songTypeAliases = []string{"parody"}
	}

	for _, songTypeAlias := range songTypeAliases {
		sequenceTmp := sequence
		if len(songTypeAlias) == 1 {
			sequenceTmp = strings.ToLower(sequence)
		} else {
			sequenceTmp = sanitize.Name(strings.ToLower(sequence))
		}
		if len(sanitize.Name(strings.ToLower(songTypeAlias))) == len(songTypeAlias) {
			songTypeAlias = sanitize.Name(strings.ToLower(songTypeAlias))
		}
		if strings.Contains(sequenceTmp, songTypeAlias) {
			return true
		}
	}
	return false
}
