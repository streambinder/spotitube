package track

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"../spotitube"
	"../system"

	"github.com/PuerkitoBio/goquery"
	"github.com/bogem/id3v2"
	"github.com/gosimple/slug"
	"github.com/mozillazg/go-unidecode"
	"github.com/zmb3/spotify"
)

// Track : struct containing all the informations about a track
type Track struct {
	Title         string
	Song          string
	Artist        string
	Album         string
	Year          string
	Featurings    []string
	Genre         string
	TrackNumber   int
	TrackTotals   int
	Duration      int
	SongType      int
	Image         string
	URL           string
	SpotifyID     string
	Filename      string
	FilenameTemp  string
	FilenameExt   string
	SearchPattern string
	Lyrics        string
	Local         bool
}

// Tracks : Track array
type Tracks []Track

// TracksDump : Tracks dumpable object
type TracksDump struct {
	Tracks Tracks
	Time   time.Time
}

// TracksIndex : Tracks index keeping ID - filename mapping and eventual filename links
type TracksIndex struct {
	Tracks map[string]string
	Links  map[string][]string
}

const (
	// GeniusAccessToken : Genius app access token
	GeniusAccessToken = ":GENIUS_TOKEN:"
	// LyricsGeniusAPIURL : lyrics Genius API URL
	LyricsGeniusAPIURL = "https://api.genius.com/search?q=%s+%s"
	// LyricsOVHAPIURL : lyrics OVH API URL
	LyricsOVHAPIURL = "https://api.lyrics.ovh/v1/%s/%s"

	// SongTypeAlbum : identifier for Song in its album variant
	SongTypeAlbum = iota
	// SongTypeLive : identifier for Song in its live variant
	SongTypeLive
	// SongTypeCover : identifier for Song in its cover variant
	SongTypeCover
	// SongTypeRemix : identifier for Song in its remix variant
	SongTypeRemix
	// SongTypeAcoustic : identifier for Song in its acoustic variant
	SongTypeAcoustic
	// SongTypeKaraoke : identifier for Song in its karaoke variant
	SongTypeKaraoke
	// SongTypeParody : identifier for Song in its parody variant
	SongTypeParody
	// SongTypeReverse : identifier for Song in its reverse variant
	SongTypeReverse
	_
	// ID3FrameTitle : ID3 title frame tag identifier
	ID3FrameTitle = iota
	// ID3FrameSong : ID3 song frame tag identifier
	ID3FrameSong
	// ID3FrameArtist : ID3 artist frame tag identifier
	ID3FrameArtist
	// ID3FrameAlbum : ID3 album frame tag identifier
	ID3FrameAlbum
	// ID3FrameGenre : ID3 genre frame tag identifier
	ID3FrameGenre
	// ID3FrameYear : ID3 year frame tag identifier
	ID3FrameYear
	// ID3FrameFeaturings : ID3 featurings frame tag identifier
	ID3FrameFeaturings
	// ID3FrameTrackNumber : ID3 track number frame tag identifier
	ID3FrameTrackNumber
	// ID3FrameTrackTotals : ID3 total tracks number frame tag identifier
	ID3FrameTrackTotals
	// ID3FrameArtwork : ID3 artwork frame tag identifier
	ID3FrameArtwork
	// ID3FrameArtworkURL : ID3 artwork URL frame tag identifier
	ID3FrameArtworkURL
	// ID3FrameLyrics : ID3 lyrics frame tag identifier
	ID3FrameLyrics
	// ID3FrameYouTubeURL : ID3 youtube URL frame tag identifier
	ID3FrameYouTubeURL
	// ID3FrameDuration : ID3 duration frame tag identifier
	ID3FrameDuration
	// ID3FrameSpotifyID : ID3 Spotify ID frame tag identifier
	ID3FrameSpotifyID
)

var (
	// SongTypes : array containing every song variant identifier
	SongTypes = []int{SongTypeLive, SongTypeCover, SongTypeRemix,
		SongTypeAcoustic, SongTypeKaraoke, SongTypeParody}
	// JunkSuffixes : array containing every file suffix considered junk
	JunkSuffixes = []string{".ytdl", ".webm", ".opus", ".part", ".jpg", ".tmp", "-id3v2"}
)

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
	if !system.FileExists(filename) {
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
		Featurings:    strings.Split(TagGetFrame(trackMp3, ID3FrameFeaturings), "|"),
		Genre:         TagGetFrame(trackMp3, ID3FrameGenre),
		TrackNumber:   0,
		TrackTotals:   0,
		Duration:      0,
		SongType:      parseType(TagGetFrame(trackMp3, ID3FrameTitle)),
		Image:         TagGetFrame(trackMp3, ID3FrameArtworkURL),
		URL:           TagGetFrame(trackMp3, ID3FrameYouTubeURL),
		SpotifyID:     TagGetFrame(trackMp3, ID3FrameSpotifyID),
		Filename:      "",
		FilenameTemp:  "",
		FilenameExt:   spotitube.SongExtension,
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
		TrackNumber: spotifyTrack.SimpleTrack.TrackNumber,
		TrackTotals: len(spotifyAlbum.Tracks.Tracks),
		Duration:    spotifyTrack.SimpleTrack.Duration / 1000,
		Image: func() string {
			if len(spotifyTrack.Album.Images) > 0 {
				return spotifyTrack.Album.Images[0].URL
			}
			return ""
		}(),
		URL:           "",
		SpotifyID:     spotifyTrack.SimpleTrack.ID.String(),
		Filename:      "",
		FilenameTemp:  "",
		FilenameExt:   spotitube.SongExtension,
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

	if system.FileExists(track.FilenameFinal()) {
		track.Local = true
	}

	if track.Local {
		track.URL = track.GetID3Frame(ID3FrameYouTubeURL)
		track.Lyrics = track.GetID3Frame(ID3FrameLyrics)
	}

	return track
}

// GetTag : open, parse and return filename ID3 tag
func GetTag(path string, frame int) string {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return ""
	}
	defer tag.Close()

	return TagGetFrame(tag, frame)
}

// FlushLocal : recheck - and eventually update it - if track is local
func (track Track) FlushLocal() Track {
	if system.FileExists(track.FilenameFinal()) {
		track.Local = true
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
		return fmt.Errorf("Item seems to be pointing to an album, not to a song")
	}
	for _, songType := range SongTypes {
		if SeemsType(sequence, songType) && track.SongType != songType {
			return fmt.Errorf("Songs seem to be of different types")
		}
	}
	return nil
}

// SeemsByWordMatch : return nil error if Track song name, artist and featurings are contained in sequence
func (track Track) SeemsByWordMatch(sequence string) error {
	sequence = slug.Make(strings.ToLower(sequence))
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
		trackItem = slug.Make(trackItem)
		if len(trackItem) > 3 && !strings.Contains(sequence, trackItem) {
			return fmt.Errorf("Songs seem to be mismatching by words comparison: \"%v+\" in \"%s\", due to \"%s\"",
				append([]string{track.Song, track.Artist}, track.Featurings...), sequence, trackItem)
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
	case ID3FrameArtworkURL:
		return TagGetFrameArtworkURL(tag)
	case ID3FrameLyrics:
		return TagGetFrameLyrics(tag)
	case ID3FrameYouTubeURL:
		return TagGetFrameYouTubeURL(tag)
	case ID3FrameDuration:
		return TagGetFrameDuration(tag)
	case ID3FrameSpotifyID:
		return TagGetFrameSpotifyID(tag)
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

// TagGetFrameFeaturings : get track featurings frame from input Tag
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

// TagGetFrameTrackTotals : get total tracks number frame from input Tag
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

// TagGetFrameArtworkURL : get artwork URL frame from input Tag
func TagGetFrameArtworkURL(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "artwork" {
				return comment.Text
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

// TagGetFrameSpotifyID : get Spotify ID frame from input Tag
func TagGetFrameSpotifyID(tag *id3v2.Tag) string {
	if len(tag.GetFrames(tag.CommonID("Comments"))) > 0 {
		for _, frameComment := range tag.GetFrames(tag.CommonID("Comments")) {
			comment, ok := frameComment.(id3v2.CommentFrame)
			if ok && comment.Description == "spotifyid" {
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
	var (
		lyrics    string
		lyricsErr error
	)
	lyrics, lyricsErr = searchLyricsGenius(track)
	if lyricsErr == nil {
		track.Lyrics = lyrics
		return nil
	}
	lyrics, lyricsErr = searchLyricsOvh(track)
	if lyricsErr == nil {
		track.Lyrics = lyrics
		return nil
	}
	return lyricsErr
}

// SeemsType : return True if input sequence matches with selected input songType variant
func SeemsType(sequence string, songType int) bool {
	var songTypeAliases []string
	if songType == SongTypeLive {
		songTypeAliases = []string{"@", "live", "perform", "tour"}
		for _, year := range system.MakeRange(1950, 2050) {
			songTypeAliases = append(songTypeAliases, []string{strconv.Itoa(year), "'" + strconv.Itoa(year)[2:]}...)
		}
	} else if songType == SongTypeCover {
		songTypeAliases = []string{"cover", "vs", "amateur"}
	} else if songType == SongTypeRemix {
		songTypeAliases = []string{"remix", "radio edit"}
	} else if songType == SongTypeAcoustic {
		songTypeAliases = []string{"acoustic"}
	} else if songType == SongTypeKaraoke {
		songTypeAliases = []string{"karaoke", "instrumental"}
	} else if songType == SongTypeParody {
		songTypeAliases = []string{"parody"}
	} else if songType == SongTypeReverse {
		songTypeAliases = []string{"reverse"}
	}

	for _, songTypeAlias := range songTypeAliases {
		sequence = strings.ToLower(sequence)
		if len(songTypeAlias) != 1 {
			sequence = slug.Make(sequence)
		}
		if len(slug.Make(strings.ToLower(songTypeAlias))) == len(songTypeAlias) {
			songTypeAlias = slug.Make(strings.ToLower(songTypeAlias))
		}
		if strings.Contains(sequence, songTypeAlias) {
			return true
		}
	}
	return false
}

func parseType(sequence string) int {
	for _, songType := range SongTypes {
		if SeemsType(sequence, songType) {
			return songType
		}
	}
	return SongTypeAlbum
}

func parseTitle(trackTitle string, trackFeaturings []string) (string, string) {
	var trackSong string

	if !(strings.Contains(trackTitle, " (") && strings.Contains(strings.Split(strings.Split(trackTitle, ")")[0], "(")[1], " - ")) {
		trackTitle = strings.Split(trackTitle, " - ")[0]
	}
	if strings.Contains(trackTitle, " live ") {
		trackTitle = strings.Split(trackTitle, " live ")[0]
	}
	trackTitle = strings.TrimSpace(trackTitle)
	if len(trackFeaturings) > 0 {
		var (
			featuringsAlreadyParsed bool
			featuringSymbols        = []string{"featuring", "feat", "ft", "with", "prod"}
		)
		for _, featuringValue := range trackFeaturings {
			for _, featuringSymbol := range featuringSymbols {
				titleParts := strings.Split(strings.ToLower(trackTitle), featuringSymbol)
				if len(titleParts) > 1 && strings.Contains(titleParts[1], strings.ToLower(featuringValue)) {
					featuringsAlreadyParsed = true
				}
			}
		}
		if featuringsAlreadyParsed {
			for _, featuringSymbol := range featuringSymbols {
				for _, featuringSymbolCase := range []string{featuringSymbol, strings.Title(featuringSymbol)} {
					trackTitle = strings.Replace(trackTitle, featuringSymbolCase+". ", "ft. ", -1)
					trackTitle = strings.Replace(trackTitle, featuringSymbolCase+" ", "ft. ", -1)
				}
			}
		} else {
			if strings.Contains(trackTitle, "(") &&
				(strings.Contains(trackTitle, " vs. ") || strings.Contains(trackTitle, " vs ")) &&
				strings.Contains(trackTitle, ")") {
				trackTitle = strings.Split(trackTitle, " (")[0]
			}
			var trackFeaturingsInline string
			if len(trackFeaturings) > 1 {
				trackFeaturingsInline = "(ft. " + strings.TrimSpace(strings.Join(trackFeaturings[:len(trackFeaturings)-1], ", ")) +
					" and " + strings.TrimSpace(trackFeaturings[len(trackFeaturings)-1]) + ")"
			} else {
				trackFeaturingsInline = "(ft. " + strings.TrimSpace(trackFeaturings[0]) + ")"
			}
			trackTitle = fmt.Sprintf("%s %s", trackTitle, trackFeaturingsInline)
		}
		trackSong = strings.Split(trackTitle, " (ft. ")[0]
	} else {
		trackSong = trackTitle
	}

	return trackTitle, trackSong
}

func parseFilename(track Track) (string, string) {
	var (
		trackFilename     string
		trackFilenameTemp string
	)
	trackFilename = track.Artist + " - " + track.Title
	for _, symbol := range []string{"/", "\\", ".", "?", "<", ">", ":", "*"} {
		trackFilename = strings.Replace(trackFilename, symbol, "", -1)
	}

	trackFilename = strings.Replace(trackFilename, "  ", " ", -1)
	trackFilename = system.Asciify(trackFilename)
	trackFilename = strings.TrimSpace(trackFilename)
	trackFilenameTemp = ("." + slug.Make(trackFilename))

	return trackFilename, trackFilenameTemp
}

func searchLyricsGenius(track *Track) (string, error) {
	var geniusToken = os.Getenv("GENIUS_TOKEN")
	if len(geniusToken) == 0 {
		geniusToken = GeniusAccessToken
	}
	if len(GeniusAccessToken) == 0 {
		return "", fmt.Errorf("Cannot fetch lyrics from Genius without a valid token")
	}

	lyricsClient := http.Client{
		Timeout: time.Second * spotitube.HTTPTimeout,
	}

	encodedURL, lyricsError := url.Parse(fmt.Sprintf(LyricsGeniusAPIURL, url.QueryEscape(track.Title), url.QueryEscape(track.Artist)))
	if lyricsError != nil {
		return "", lyricsError
	}
	lyricsRequest, lyricsError := http.NewRequest(http.MethodGet, encodedURL.String(), nil)
	if lyricsError != nil {
		return "", lyricsError
	}
	lyricsRequest.Header.Add("Authorization", fmt.Sprintf("Bearer %s", GeniusAccessToken))

	lyricsResponse, lyricsError := lyricsClient.Do(lyricsRequest)
	if lyricsError != nil {
		return "", lyricsError
	}

	lyricsResponseBody, lyricsError := ioutil.ReadAll(lyricsResponse.Body)
	if lyricsError != nil {
		return "", lyricsError
	}

	var result map[string]interface{}
	hitsUnmarshalErr := json.Unmarshal([]byte(lyricsResponseBody), &result)
	if hitsUnmarshalErr != nil {
		return "", hitsUnmarshalErr
	}

	hits := result["response"].(map[string]interface{})["hits"].([]interface{})
	var lyricsURL string
	for _, value := range hits {
		valueResult := value.(map[string]interface{})["result"].(map[string]interface{})
		songTitle := strings.TrimSpace(valueResult["title"].(string))
		songArtist := strings.TrimSpace(valueResult["primary_artist"].(map[string]interface{})["name"].(string))

		songErr := track.Seems(fmt.Sprintf("%s %s", songTitle, songArtist))
		if songErr == nil {
			lyricsURL = strings.TrimSpace(valueResult["url"].(string))
			break
		}
	}

	if len(lyricsURL) == 0 {
		return "", fmt.Errorf("Genius lyrics not found")
	}

	doc, _ := goquery.NewDocument(lyricsURL)
	return strings.TrimSpace(unidecode.Unidecode(doc.Find(".lyrics").Eq(0).Text())), nil
}

func searchLyricsOvh(track *Track) (string, error) {
	type LyricsAPIEntry struct {
		Lyrics string `json:"lyrics"`
	}
	lyricsClient := http.Client{
		Timeout: time.Second * spotitube.HTTPTimeout,
	}

	encodedURL, lyricsError := url.Parse(fmt.Sprintf(LyricsOVHAPIURL, url.QueryEscape(track.Artist), url.QueryEscape(track.Song)))
	if lyricsError != nil {
		return "", lyricsError
	}
	lyricsRequest, lyricsError := http.NewRequest(http.MethodGet, encodedURL.String(), nil)
	if lyricsError != nil {
		return "", lyricsError
	}
	lyricsResponse, lyricsError := lyricsClient.Do(lyricsRequest)
	if lyricsError != nil {
		return "", lyricsError
	}
	lyricsResponseBody, lyricsError := ioutil.ReadAll(lyricsResponse.Body)
	if lyricsError != nil {
		return "", lyricsError
	}
	lyricsData := LyricsAPIEntry{}
	lyricsError = json.Unmarshal(lyricsResponseBody, &lyricsData)
	if lyricsError != nil {
		return "", lyricsError
	}

	return strings.TrimSpace(unidecode.Unidecode(lyricsData.Lyrics)), nil
}
