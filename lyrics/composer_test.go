package lyrics

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
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
	assert.Equal(t, chooseComposition(nil, []byte("lyrics")), []byte("lyrics"))
	assert.Equal(t, chooseComposition([]byte("[00:27.37]lyrics"), []byte("lyrics")), []byte("[00:27.37]lyrics"))
	assert.Equal(t, chooseComposition([]byte("lyrics"), []byte("[00:27.37]lyrics")), []byte("[00:27.37]lyrics"))
	assert.Equal(t, chooseComposition([]byte("lyrics"), []byte("lyrics but longer")), []byte("lyrics but longer"))
}

func TestSearch(t *testing.T) {
	// monkey patching
	ch := make(chan bool, 1)
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) { return nil, errors.New("") }).
		ApplyPrivateMethod(reflect.TypeOf(genius{}), "search", func() ([]byte, error) {
			close(ch)
			return []byte("glyrics"), nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(lrclib{}), "search", func() ([]byte, error) {
			<-ch
			return []byte("[00:27.37]llyrics"), nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "search", func() ([]byte, error) {
			<-ch
			return []byte("olyrics"), nil
		}).
		Reset()

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Equal(t, "[00:27.37]llyrics", lyrics)
}

func TestSearchAlreadyExists(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(os.ReadFile, func() ([]byte, error) {
		return []byte("lyrics"), nil
	}).Reset()

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Equal(t, "lyrics", lyrics)
}

func TestSearchFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return nil, errors.New("")
		}).
		ApplyPrivateMethod(reflect.TypeOf(genius{}), "search", func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		ApplyPrivateMethod(reflect.TypeOf(lrclib{}), "search", func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "search", func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, sys.ErrOnly(Search(track)), "ko")
}

func TestSearchNotFound(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return nil, errors.New("")
		}).
		ApplyPrivateMethod(reflect.TypeOf(genius{}), "search", func() ([]byte, error) {
			return nil, nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(lrclib{}), "search", func() ([]byte, error) {
			return nil, nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "search", func() ([]byte, error) {
			return nil, nil
		}).
		Reset()

	// testing
	lyrics, err := Search(track)
	assert.Nil(t, err)
	assert.Empty(t, lyrics)
}

func TestSearchCannotCreateDir(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return nil, errors.New("")
		}).
		ApplyPrivateMethod(reflect.TypeOf(genius{}), "search", func() ([]byte, error) {
			return []byte("lyrics"), nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(lrclib{}), "search", func() ([]byte, error) {
			return []byte{}, nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "search", func() ([]byte, error) {
			return []byte{}, nil
		}).
		ApplyFunc(os.MkdirAll, func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, sys.ErrOnly(Search(track)), "ko")
}

func TestGet(t *testing.T) {
	// monkey patching
	ch := make(chan bool, 1)
	defer gomonkey.NewPatches().
		ApplyPrivateMethod(reflect.TypeOf(genius{}), "get", func() ([]byte, error) {
			close(ch)
			return []byte("glyrics"), nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(lrclib{}), "get", func() ([]byte, error) {
			<-ch
			return []byte("[00:27.37]llyrics"), nil
		}).
		ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "get", func() ([]byte, error) {
			<-ch
			return []byte("olyrics"), nil
		}).
		Reset()

	// testing
	lyrics, err := Get("http://localhost")
	assert.Nil(t, err)
	assert.Equal(t, "[00:27.37]llyrics", lyrics)
}

func TestGetFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyPrivateMethod(reflect.TypeOf(genius{}), "get", func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		ApplyPrivateMethod(reflect.TypeOf(lrclib{}), "get", func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		ApplyPrivateMethod(reflect.TypeOf(lyricsOvh{}), "get", func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, sys.ErrOnly(Get("http://localhost")), "ko")
}
