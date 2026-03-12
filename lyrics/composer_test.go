package lyrics

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys"
	"github.com/stretchr/testify/assert"
)

var track = &entity.Track{
	Title:   "Title",
	Artists: []string{"Artist"},
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
	// monkey patching
	ch := make(chan bool, 1)
	defer mockey.UnPatchAll()
	mockey.Mock(os.ReadFile).Return(nil, errors.New("")).Build()
	mockey.Mock(mockey.GetMethod(genius{}, "search")).To(func(_ genius, _ *entity.Track, _ ...context.Context) ([]byte, error) {
		close(ch)
		return []byte("glyrics"), nil
	}).Build()
	mockey.Mock(mockey.GetMethod(lrclib{}, "search")).To(func(_ lrclib, _ *entity.Track, _ ...context.Context) ([]byte, error) {
		<-ch
		return []byte("[00:27.37]llyrics"), nil
	}).Build()
	mockey.Mock(mockey.GetMethod(lyricsOvh{}, "search")).To(func(_ lyricsOvh, _ *entity.Track, _ ...context.Context) ([]byte, error) {
		<-ch
		return []byte("olyrics"), nil
	}).Build()

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Equal(t, "[00:27.37]llyrics", lyrics)
}

func TestSearchAlreadyExists(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.ReadFile).Return([]byte("lyrics"), nil).Build()

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Equal(t, "lyrics", lyrics)
}

func TestSearchFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.ReadFile).Return(nil, errors.New("")).Build()
	mockey.Mock(mockey.GetMethod(genius{}, "search")).Return(nil, errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(lrclib{}, "search")).Return(nil, errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(lyricsOvh{}, "search")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(Search(track)), "ko")
}

func TestSearchNotFound(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.ReadFile).Return(nil, errors.New("")).Build()
	mockey.Mock(mockey.GetMethod(genius{}, "search")).Return(nil, nil).Build()
	mockey.Mock(mockey.GetMethod(lrclib{}, "search")).Return(nil, nil).Build()
	mockey.Mock(mockey.GetMethod(lyricsOvh{}, "search")).Return(nil, nil).Build()

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Empty(t, lyrics)
}

func TestSearchCannotCreateDir(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.ReadFile).Return(nil, errors.New("")).Build()
	mockey.Mock(mockey.GetMethod(genius{}, "search")).Return([]byte("lyrics"), nil).Build()
	mockey.Mock(mockey.GetMethod(lrclib{}, "search")).Return([]byte{}, nil).Build()
	mockey.Mock(mockey.GetMethod(lyricsOvh{}, "search")).Return([]byte{}, nil).Build()
	mockey.Mock(os.MkdirAll).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(Search(track)), "ko")
}

func TestSearchWriteFileFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(os.ReadFile).Return(nil, errors.New("")).Build()
	mockey.Mock(mockey.GetMethod(genius{}, "search")).Return([]byte("lyrics"), nil).Build()
	mockey.Mock(mockey.GetMethod(lrclib{}, "search")).Return([]byte{}, nil).Build()
	mockey.Mock(mockey.GetMethod(lyricsOvh{}, "search")).Return([]byte{}, nil).Build()
	mockey.Mock(os.MkdirAll).Return(nil).Build()
	mockey.Mock(os.WriteFile).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(Search(track)), "ko")
}

func TestGet(t *testing.T) {
	// monkey patching
	ch := make(chan bool, 1)
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(genius{}, "get")).To(func(_ genius, _ string, _ ...context.Context) ([]byte, error) {
		close(ch)
		return []byte("glyrics"), nil
	}).Build()
	mockey.Mock(mockey.GetMethod(lrclib{}, "get")).To(func(_ lrclib, _ string, _ ...context.Context) ([]byte, error) {
		<-ch
		return []byte("[00:27.37]llyrics"), nil
	}).Build()
	mockey.Mock(mockey.GetMethod(lyricsOvh{}, "get")).To(func(_ lyricsOvh, _ string, _ ...context.Context) ([]byte, error) {
		<-ch
		return []byte("olyrics"), nil
	}).Build()

	// testing
	lyrics, err := Get("http://localhost")
	assert.Nil(t, err)
	assert.Equal(t, "[00:27.37]llyrics", lyrics)
}

func TestGetFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(mockey.GetMethod(genius{}, "get")).Return(nil, errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(lrclib{}, "get")).Return(nil, errors.New("ko")).Build()
	mockey.Mock(mockey.GetMethod(lyricsOvh{}, "get")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(Get("http://localhost")), "ko")
}
