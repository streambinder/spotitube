package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/adrg/xdg"
	"github.com/bogem/id3v2/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/entity/id3"
	"github.com/streambinder/spotitube/entity/index"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
)

// auditTrack couples a fetched track with the library selector
// it was fetched from, so a collision can be attributed to its source
type auditTrack struct {
	track  *entity.Track
	source string
}

// auditCollision is a single filename collision found by the audit:
// the incoming fetched track would share its filename with an already
// indexed track, which is what aborts a sync
type auditCollision struct {
	filename string
	incoming auditTrack
	existing auditTrack
}

func init() {
	cmdRoot.AddCommand(cmdAudit())
}

// flagLibrary is the library selector flag
const flagLibrary = "library"

// collectionSelection is the set of library selectors read from the flags
type collectionSelection struct {
	library         bool
	playlists       []string
	playlistsTracks []string
	albums          []string
	tracks          []string
	libraryLimit    int
}

// readSelection reads the collection selector flags
func readSelection(cmd *cobra.Command) collectionSelection {
	return collectionSelection{
		library:         sys.ErrWrap(false)(cmd.Flags().GetBool(flagLibrary)),
		playlists:       sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("playlist")),
		playlistsTracks: sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("playlist-tracks")),
		albums:          sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("album")),
		tracks:          sys.ErrWrap([]string{})(cmd.Flags().GetStringArray("track")),
		libraryLimit:    sys.ErrWrap(0)(cmd.Flags().GetInt("library-limit")),
	}
}

// enableLibraryDefault turns the library selector on when no collection
// selector was supplied
func enableLibraryDefault(cmd *cobra.Command, supplied int) {
	if supplied == 0 {
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if f.Name == flagLibrary {
				sys.ErrSuppress(f.Value.Set("true"))
			}
		})
	}
}

func cmdAudit() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "audit",
		Short:        "Check the library for filename collisions without downloading anything",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var (
				path = sys.ErrWrap(xdg.UserDirs.Music)(cmd.Flags().GetString("output"))
				sel  = readSelection(cmd)
			)

			if err := os.Chdir(path); err != nil {
				return err
			}

			// the audit is a plain sequential run: index the library,
			// authenticate, fetch every selected collection, then check each
			// fetched track for filename collisions instead of aborting on
			// the first one. Nothing is downloaded.
			// The report is printed only if every step succeeded: a fetch
			// failure must never look like a clean audit.
			if err := auditBuildIndex(); err != nil {
				return err
			}
			if err := auditAuthenticate(); err != nil {
				return err
			}
			fetched, err := auditFetch(sel)
			if err != nil {
				return err
			}

			results := &auditResults{}
			claimants := map[string]auditTrack{}
			for _, incoming := range fetched {
				results.checked++
				if collides, claims := auditClassify(incoming.track); collides {
					results.collisions = append(results.collisions, resolveAuditCollision(incoming, claimants))
				} else if claims {
					claimants[incoming.track.Path().Final()] = incoming
				}
			}
			fmt.Printf("checked %d tracks\n", results.checked)

			return results.report()
		},
		PreRun: func(cmd *cobra.Command, _ []string) {
			sel := readSelection(cmd)
			enableLibraryDefault(cmd, len(sel.playlists)+len(sel.playlistsTracks)+len(sel.albums)+len(sel.tracks))
		},
	}
	cmd.Flags().StringP("output", "o", xdg.UserDirs.Music, "Library path to audit")
	cmd.Flags().BoolP(flagLibrary, "l", false, "Audit library (auto-enabled if no collection is supplied)")
	cmd.Flags().StringArrayP("playlist", "p", []string{}, "Audit playlist")
	cmd.Flags().StringArray("playlist-tracks", []string{}, "Audit playlist tracks without playlist file")
	cmd.Flags().StringArrayP("album", "a", []string{}, "Audit album")
	cmd.Flags().StringArrayP("track", "t", []string{}, "Audit track")
	cmd.Flags().Int("library-limit", 0, "Number of tracks to fetch from library (unlimited if 0)")
	return cmd
}

// auditBuildIndex scans the local music library with plain terminal output
func auditBuildIndex() error {
	fmt.Println("indexing library...")

	indexed := make(chan string)
	var counter sync.WaitGroup
	counter.Add(1)
	go func() {
		defer counter.Done()
		count := 0
		for range indexed {
			count++
		}
		fmt.Printf("indexed %d tracks\n", count)
	}()

	err := indexData.BuildWithProgress(".", indexed)
	close(indexed)
	counter.Wait()
	return err
}

// auditAuthenticate resolves the Spotify client with plain terminal output
func auditAuthenticate() error {
	var err error
	spotifyClient, err = spotify.Authenticate(spotify.BrowserProcessor)
	if err != nil {
		return err
	}
	defer spotifyClient.Close()

	if username, err := spotifyClient.Username(); err != nil {
		fmt.Println("could not resolve authenticated username:", err)
	} else {
		fmt.Println("authenticated as", username)
	}
	return nil
}

// auditFetch pulls data from the upstream provider like the sync fetcher does,
// tagging every track with the library selector it was fetched from
// (library, playlist, playlist-tracks, album or track), so collisions can be
// attributed to their source. The audit is strictly read-only and never
// touches the library.
func auditFetch(sel collectionSelection) ([]auditTrack, error) {
	var fetched []auditTrack

	// collector runs a single source fetch, tagging every track with the
	// selector it was fetched from
	collector := func(source string, fetch func(ch ...chan interface{}) error) error {
		tagged := make(chan interface{}, pipelineBuffer)
		var collectorWait sync.WaitGroup
		collectorWait.Add(1)
		go func() {
			defer collectorWait.Done()
			for event := range tagged {
				fetched = append(fetched, auditTrack{track: event.(*entity.Track), source: source})
			}
		}()
		err := fetch(tagged)
		close(tagged)
		collectorWait.Wait()
		if err != nil {
			return fmt.Errorf("%s: %w", source, err)
		}
		return nil
	}

	if sel.library {
		fmt.Println("fetching library")
		if err := collector(flagLibrary, func(ch ...chan interface{}) error {
			return spotifyClient.Library(sel.libraryLimit, ch...)
		}); err != nil {
			return nil, err
		}
	}
	for _, id := range sel.albums {
		fmt.Println("fetching album", id)
		if err := collector("album "+id, func(ch ...chan interface{}) error {
			_, err := spotifyClient.Album(id, ch...)
			return err
		}); err != nil {
			return nil, err
		}
	}
	for _, id := range sel.tracks {
		fmt.Println("fetching track", id)
		if err := collector("track "+id, func(ch ...chan interface{}) error {
			_, err := spotifyClient.Track(id, ch...)
			return err
		}); err != nil {
			return nil, err
		}
	}
	for _, name := range sel.playlists {
		fmt.Println("fetching playlist", name)
		if err := collector("playlist "+name, func(ch ...chan interface{}) error {
			_, err := spotifyClient.Playlist(name, ch...)
			return err
		}); err != nil {
			return nil, err
		}
	}
	for _, name := range sel.playlistsTracks {
		fmt.Println("fetching playlist", name)
		if err := collector("playlist-tracks "+name, func(ch ...chan interface{}) error {
			_, err := spotifyClient.Playlist(name, ch...)
			return err
		}); err != nil {
			return nil, err
		}
	}

	return fetched, nil
}

// auditResults collects the outcome of an audit run: every collision found
// and how many tracks were checked
type auditResults struct {
	collisions []auditCollision
	checked    int
}

// report prints every collision found with the source of both sides,
// or confirms a clean audit. It must run only after every step completed
// without errors.
func (results *auditResults) report() error {
	if len(results.collisions) == 0 {
		fmt.Println("no filename collisions found")
		return nil
	}

	sort.Slice(results.collisions, func(i, j int) bool { return results.collisions[i].filename < results.collisions[j].filename })

	fmt.Printf(colorRed+"%d filename collisions found:\n"+colorReset, len(results.collisions))
	for _, collision := range results.collisions {
		fmt.Printf("collision: %q\n", collision.filename)
		fmt.Printf("  incoming %q by %q (spotify id %s) from %s\n",
			collision.incoming.track.Title, collision.incoming.track.Artist(),
			collision.incoming.track.ID, collision.incoming.source)
		fmt.Printf("  existing %q by %q (spotify id %s) from %s\n",
			collision.existing.track.Title, collision.existing.track.Artist(),
			collision.existing.track.ID, collision.existing.source)
	}
	return fmt.Errorf("audit failed: %d filename collisions found", len(results.collisions))
}

// auditClassify mirrors decideClassify's collision predicate as a dry run:
// it reports whether the track would abort a sync with a filename collision
// and whether it claimed the filename in the index, so that a later
// same-filename track is attributed to it instead of being missed.
// It must stay aligned with decideClassify: any change to the collision
// condition there applies here as well.
func auditClassify(track *entity.Track) (collides, claims bool) {
	decideIndexMu.Lock()
	defer decideIndexMu.Unlock()

	var (
		idStatus int
		idKnown  bool
	)
	if len(track.ID) > 0 {
		idStatus, idKnown = indexData.GetID(track.ID)
	}
	_, pathKnown := indexData.GetPath(track.Path().Final())

	// a track unknown to the index whose filename is already taken would
	// abort a sync: report the collision without claiming the filename
	if !idKnown && pathKnown {
		return true, false
	}
	// a track that would proceed with the sync claims its filename as online,
	// so that a later track resolving to the same filename is reported as
	// colliding with it instead of being missed
	if !idKnown || idStatus == index.Flush {
		indexData.Set(track, index.Online)
		return false, true
	}
	return false, false
}

// resolveAuditCollision builds the full collision entry: the incoming track
// is known, while the existing claimant is either another track fetched
// during this audit or the library file itself, in which case its metadata
// is read back from the ID3 tag
func resolveAuditCollision(incoming auditTrack, claimants map[string]auditTrack) auditCollision {
	filename := incoming.track.Path().Final()
	collision := auditCollision{filename: filename, incoming: incoming}

	if existing, ok := claimants[filename]; ok {
		collision.existing = existing
		return collision
	}

	collision.existing = auditTrack{
		source: flagLibrary,
		track:  &entity.Track{Title: filepath.Base(filename)},
	}
	if tag, err := id3.Open(filename, id3v2.Options{Parse: true}); err == nil {
		collision.existing.track = &entity.Track{
			ID:      tag.SpotifyID(),
			Title:   tag.Title(),
			Artists: []string{tag.Artist()},
		}
		_ = tag.Close()
	}
	return collision
}
