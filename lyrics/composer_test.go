package lyrics

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

var track = &entity.Track{
	Title:   "Title",
	Artists: []string{"Artist"},
}

// stubComposer is a scripted Composer: the package-level composers
// slice is swapped for the duration of a test, so Search and Get run
// against known composer outcomes.
type stubComposer struct {
	searchResult []byte
	searchErr    error
	getResult    []byte
	getErr       error
}

func (s stubComposer) search(*entity.Track, ...context.Context) ([]byte, error) {
	return s.searchResult, s.searchErr
}

func (s stubComposer) get(string, ...context.Context) ([]byte, error) {
	return s.getResult, s.getErr
}

func setComposers(t *testing.T, stubs ...Composer) {
	t.Helper()
	previous := composers
	composers = stubs
	t.Cleanup(func() { composers = previous })
}

// uncachedTrack returns a track whose lyrics cache file is guaranteed
// absent, and removes the file again at the end of the test.
func uncachedTrack(t *testing.T, id string) *entity.Track {
	t.Helper()
	uncached := &entity.Track{ID: id, Title: "Title", Artists: []string{"Artist"}}
	path := uncached.Path().Lyrics()
	_ = os.Remove(path)
	t.Cleanup(func() { _ = os.Remove(path) })
	return uncached
}

func BenchmarkComposer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestSearch(&testing.T{})
	}
}

func TestIsSynced(t *testing.T) {
	assert.True(t, IsSynced([]byte("[00:27.37]lyrics")))
	assert.False(t, IsSynced([]byte("lyrics")))
	assert.True(t, IsSynced("[00:27.37]lyrics"))
	assert.False(t, IsSynced("lyrics"))
	assert.False(t, IsSynced(123))
}

func TestGetPlain(t *testing.T) {
	assert.Equal(t, GetPlain("[00:27.37]lyrics"), "[00:27.37]lyrics")
	assert.Equal(t, GetPlain("lyrics"), "lyrics")
}

func TestGetSync(t *testing.T) {
	assert.Equal(t, GetSync("[00:27.37]lyrics"), []SyncedLine{{27370, "lyrics"}})
	assert.Equal(t, GetSync("[00:27.37]lyrics\n[00:27.37]"), []SyncedLine{{27370, "lyrics"}})
	assert.Equal(t, GetSync("lyrics"), []SyncedLine{})
}

func TestChooseComposition(t *testing.T) {
	assert.Nil(t, chooseComposition(nil, nil))
	assert.Equal(t, []byte("lyrics"), chooseComposition([]byte("lyrics"), nil))
	assert.Equal(t, []byte("lyrics"), chooseComposition(nil, []byte("lyrics")))
	assert.Equal(t, []byte("[00:27.37]lyrics"), chooseComposition([]byte("[00:27.37]lyrics"), []byte("lyrics")))
	assert.Equal(t, []byte("[00:27.37]lyrics"), chooseComposition([]byte("lyrics"), []byte("[00:27.37]lyrics")))
	assert.Equal(t, []byte("lyrics but longer"), chooseComposition([]byte("lyrics but longer"), []byte("lyrics")))
	assert.Equal(t, []byte("lyrics but longer"), chooseComposition([]byte("lyrics"), []byte("lyrics but longer")))
}

func TestSearch(t *testing.T) {
	uncached := uncachedTrack(t, "test-search")
	setComposers(t,
		stubComposer{searchResult: []byte("glyrics")},
		stubComposer{searchResult: []byte("[00:27.37]llyrics")},
	)

	// testing: the synced composition wins over the plain one
	lyrics, err := Search(uncached)
	assert.Nil(t, err)
	assert.Equal(t, "[00:27.37]llyrics", lyrics)
}

func TestSearchAlreadyExists(t *testing.T) {
	cached := uncachedTrack(t, "test-cached")
	assert.Nil(t, os.MkdirAll(filepath.Dir(cached.Path().Lyrics()), 0o755))
	assert.Nil(t, os.WriteFile(cached.Path().Lyrics(), []byte("lyrics"), 0o600))
	setComposers(t, stubComposer{searchErr: errors.New("must not be called")})

	// testing: the cached lyrics are returned without any composer
	lyrics, err := Search(cached)
	assert.Nil(t, err)
	assert.Equal(t, "lyrics", lyrics)
}

func TestSearchFailure(t *testing.T) {
	uncached := uncachedTrack(t, "test-failure")
	setComposers(t,
		stubComposer{searchErr: errors.New("ko")},
		stubComposer{searchErr: errors.New("ko")},
	)

	// testing: every composer failed → the first failure is returned
	assert.EqualError(t, sys.ErrOnly(Search(uncached)), "ko")
}

func TestSearchNotFound(t *testing.T) {
	uncached := uncachedTrack(t, "test-notfound")
	setComposers(t, stubComposer{}, stubComposer{})

	// testing
	lyrics, err := Search(uncached)
	assert.Nil(t, err)
	assert.Empty(t, lyrics)
}

func TestSearchCannotCreateDir(t *testing.T) {
	uncached := uncachedTrack(t, "test-nodir")
	setComposers(t,
		stubComposer{searchResult: []byte("lyrics")},
		stubComposer{searchResult: []byte{}},
	)

	// point the cache home at a regular file, so creating the lyrics
	// cache directory must fail
	blocker := filepath.Join(t.TempDir(), "blocker")
	assert.Nil(t, os.WriteFile(blocker, []byte{}, 0o644))
	t.Cleanup(func() { xdg.Reload() })
	t.Setenv("XDG_CACHE_HOME", blocker)
	xdg.Reload()

	// testing
	assert.NotNil(t, sys.ErrOnly(Search(uncached)))
}

func TestSearchWriteFileFailure(t *testing.T) {
	blocked := uncachedTrack(t, "test-nowrite")
	setComposers(t,
		stubComposer{searchResult: []byte("lyrics")},
		stubComposer{searchResult: []byte{}},
	)

	// a directory occupies the lyrics cache file path
	assert.Nil(t, os.MkdirAll(blocked.Path().Lyrics(), 0o755))
	t.Cleanup(func() { _ = os.RemoveAll(blocked.Path().Lyrics()) })

	// testing
	assert.NotNil(t, sys.ErrOnly(Search(blocked)))
}

func TestGet(t *testing.T) {
	setComposers(t,
		stubComposer{getResult: []byte("glyrics")},
		stubComposer{getResult: []byte("[00:27.37]llyrics")},
	)

	// testing: the synced composition wins over the plain one
	lyrics, err := Get("http://localhost")
	assert.Nil(t, err)
	assert.Equal(t, "[00:27.37]llyrics", lyrics)
}

func TestGetFailure(t *testing.T) {
	setComposers(t,
		stubComposer{getErr: errors.New("ko")},
		stubComposer{getErr: errors.New("ko")},
	)

	// testing
	assert.EqualError(t, sys.ErrOnly(Get("http://localhost")), "ko")
}

func TestSearchPartialFailure(t *testing.T) {
	uncached := uncachedTrack(t, "test-partial")
	setComposers(t,
		stubComposer{searchErr: errors.New("ko")},
		stubComposer{searchResult: []byte("lyrics")},
	)

	// testing: one failing composer must not abort the others
	lyrics, err := Search(uncached)
	assert.Nil(t, err)
	assert.Equal(t, "lyrics", lyrics)
}

func TestSearchPartialFailureNotFound(t *testing.T) {
	uncached := uncachedTrack(t, "test-partial-notfound")
	setComposers(t,
		stubComposer{},
		stubComposer{searchErr: errors.New("ko")},
	)

	// testing: a not-found composer plus a failing one is not a failure
	lyrics, err := Search(uncached)
	assert.Nil(t, err)
	assert.Empty(t, lyrics)
}

func TestGetPartialFailure(t *testing.T) {
	setComposers(t,
		stubComposer{getErr: errors.New("ko")},
		stubComposer{getResult: []byte("lyrics")},
	)

	// testing: one failing composer must not abort the others
	lyrics, err := Get("http://localhost")
	assert.Nil(t, err)
	assert.Equal(t, "lyrics", lyrics)
}
