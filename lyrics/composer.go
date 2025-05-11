package lyrics

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/arunsworld/nursery"
	"github.com/streambinder/spotitube/entity"
)

var (
	composers      = []Composer{}
	reSyncedPrefix = regexp.MustCompile(`^\[\d{2}:\d{2}\.\d{2}\]\s*`)
)

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
	return reSyncedPrefix.MatchString(strings.Split(lyrics, "\n")[0])
}

func GetPlain(lyrics string) string {
	if IsSynced(lyrics) {
		lines := strings.Split(lyrics, "\n")
		for i, line := range lines {
			lines[i] = reSyncedPrefix.ReplaceAllString(line, "")
		}
		return strings.Join(lines, "\n")

	}
	return lyrics
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
