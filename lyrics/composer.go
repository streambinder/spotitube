package lyrics

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys"
)

var (
	composers    = []Composer{}
	reSyncedLine = regexp.MustCompile(`^\[(\d{2}:\d{2}\.\d{2})\]\s*(.+)`)
)

type SyncedLine struct {
	Time uint32
	Text string
}

type Composer interface {
	search(*entity.Track, ...context.Context) ([]byte, error)
	get(string, ...context.Context) ([]byte, error)
}

func IsSynced(data interface{}) bool {
	var lyrics string
	switch v := data.(type) {
	case string:
		lyrics = v
	case []byte:
		lyrics = string(v)
	default:
		return false
	}
	return reSyncedLine.MatchString(strings.Split(lyrics, "\n")[0])
}

func GetPlain(lyrics string) string {
	// In an ideal world, standards are standards and everyone follows them.
	// Unfortunately, the world is not ideal, hence the USLT and SYLT ID3 tags
	// aren't the standard way of distinguishing between plain and synced lyrics.
	// Rather, it's enough for the USLT tag to be in LRC format to be considered
	// synced. I'll leave the ideal logic commented out for future reference.

	// if !IsSynced(lyrics) {
	// 	return lyrics
	// }

	// lines := strings.Split(lyrics, "\n")
	// for i, line := range lines {
	// 	if matches := reSyncedLine.FindStringSubmatch(line); len(matches) == 3 {
	// 		lines[i] = matches[2]
	// 	}
	// }
	// return strings.Join(lines, "\n")
	return lyrics
}

func GetSync(lyrics string) []SyncedLine {
	if !IsSynced(lyrics) {
		return []SyncedLine{}
	}

	var lines []SyncedLine
	for _, line := range strings.Split(lyrics, "\n") {
		matches := reSyncedLine.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}

		lines = append(lines, SyncedLine{
			// since we're regex-testing the line first, we know for sure
			// that the time is in the format we expect
			Time: sys.ErrWrap(uint32(0))(sys.ColonMinutesToMillis(matches[1])),
			Text: matches[2],
		})
	}
	return lines
}

func chooseComposition(first, latter []byte) []byte {
	switch {
	case first == nil && latter == nil:
		return nil
	case first != nil && latter == nil:
		return first
	case first == nil && latter != nil:
		return latter
	case IsSynced(first) && !IsSynced(latter):
		return first
	case !IsSynced(first) && IsSynced(latter):
		return latter
	case len(first) > len(latter):
		return first
	default:
		return latter
	}
}

// not found entries return no error
func Search(track *entity.Track) (string, error) {
	if bytes, err := os.ReadFile(track.Path().Lyrics()); err == nil {
		return string(bytes), nil
	}

	var (
		workers        []nursery.ConcurrentJob
		result         []byte
		ctxBackground  = context.Background()
		ctx, ctxCancel = context.WithCancel(ctxBackground)
	)
	defer ctxCancel()

	for _, composer := range composers {
		workers = append(workers, func(c Composer) func(context.Context, chan error) {
			return func(ctx context.Context, ch chan error) {
				lyrics, err := c.search(track, ctx)
				if err != nil {
					ch <- err
					return
				}

				if choice := chooseComposition(lyrics, result); choice != nil {
					result = choice
					if IsSynced(choice) {
						ctxCancel()
					}
				}
			}
		}(composer))
	}

	if err := nursery.RunConcurrentlyWithContext(ctx, workers...); err != nil {
		return "", err
	}

	if len(result) == 0 {
		return "", nil
	}

	if err := os.MkdirAll(filepath.Dir(track.Path().Lyrics()), os.ModePerm); err != nil {
		return "", err
	}

	return string(result), os.WriteFile(track.Path().Lyrics(), result, 0o600)
}

func Get(url string) (string, error) {
	var (
		workers        []nursery.ConcurrentJob
		result         []byte
		ctxBackground  = context.Background()
		ctx, ctxCancel = context.WithCancel(ctxBackground)
	)
	defer ctxCancel()

	for _, composer := range composers {
		workers = append(workers, func(c Composer) func(context.Context, chan error) {
			return func(ctx context.Context, ch chan error) {
				lyrics, err := c.get(url, ctx)
				if err != nil {
					ch <- err
					return
				}

				if choice := chooseComposition(lyrics, result); choice != nil {
					result = choice
					if IsSynced(choice) {
						ctxCancel()
					}
				}
			}
		}(composer))
	}

	return string(result), nursery.RunConcurrentlyWithContext(ctx, workers...)
}
