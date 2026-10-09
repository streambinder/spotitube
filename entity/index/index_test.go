package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bogem/id3v2/v2"
	"github.com/streambinder/spotitube/entity"
	"github.com/stretchr/testify/assert"
)

func BenchmarkIndex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestBuild(&testing.T{})
	}
}

// writeTrackFile creates a file holding a real ID3v2 tag and returns
// its path. When spotifyID is not empty, the tag carries it as a
// user-defined text frame, the way a synced track would.
func writeTrackFile(t *testing.T, path, spotifyID string) {
	t.Helper()

	tag := id3v2.NewEmptyTag()
	if len(spotifyID) > 0 {
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    tag.DefaultEncoding(),
			Description: "Spotify ID",
			Value:       spotifyID,
		})
	}

	file, err := os.Create(path)
	assert.Nil(t, err)
	_, err = tag.WriteTo(file)
	assert.Nil(t, err)
	assert.Nil(t, file.Close())
}

// writeFile creates a file with the given contents and returns its path
func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	assert.Nil(t, os.WriteFile(path, data, 0o600))
}

func TestBuild(t *testing.T) {
	// testing: a tree holding a synced track, an untagged file, an
	// unsupported file, an unreadable track and a nested directory:
	// only the synced track at the root level gets indexed
	root := t.TempDir()
	writeTrackFile(t, filepath.Join(root, "Artist - Title.mp3"), "id")
	writeTrackFile(t, filepath.Join(root, "Artist - Untagged.mp3"), "")
	writeFile(t, filepath.Join(root, "notes.txt"), []byte("not a track"))
	assert.Nil(t, os.Symlink(
		filepath.Join(root, "missing.mp3"),
		filepath.Join(root, "Artist - Broken.mp3"),
	))
	subdir := filepath.Join(root, "subdir")
	assert.Nil(t, os.Mkdir(subdir, 0o755))
	writeTrackFile(t, filepath.Join(subdir, "Artist - Inner.mp3"), "inner")

	index := New()
	index.Set(&entity.Track{ID: "id", Title: "Title", Artists: []string{"Artist"}}, Offline)
	assert.Nil(t, index.Build(root, 0))
	status, ok := index.Get(&entity.Track{ID: "id", Artists: []string{"Artist"}, Title: "Title"})
	assert.True(t, ok)
	assert.Equal(t, 0, status)
	assert.Equal(t, 1, index.Size())
	assert.Equal(t, 1, index.Size(Offline))

	// the track in the nested directory is skipped along with the directory
	_, ok = index.GetID("inner")
	assert.False(t, ok)
}

func TestBuildWithProgress(t *testing.T) {
	// testing
	root := t.TempDir()
	writeTrackFile(t, filepath.Join(root, "Artist - Title.mp3"), "id")
	writeTrackFile(t, filepath.Join(root, "Artist - Song.mp3"), "id")

	indexed := make(chan string, 10)
	index := New()
	assert.Nil(t, index.BuildWithProgress(root, indexed))
	close(indexed)

	var paths []string
	for p := range indexed {
		paths = append(paths, p)
	}
	assert.Len(t, paths, 2)
	assert.ElementsMatch(t, []string{
		filepath.Join(root, "Artist - Title.mp3"),
		filepath.Join(root, "Artist - Song.mp3"),
	}, paths)
}

func TestBuildSkipsUnreadableFiles(t *testing.T) {
	// testing: a track that cannot be opened (a dangling symlink) is
	// skipped, the walk continues
	root := t.TempDir()
	assert.Nil(t, os.Symlink(
		filepath.Join(root, "missing.mp3"),
		filepath.Join(root, "broken.mp3"),
	))
	writeTrackFile(t, filepath.Join(root, "Artist - Title.mp3"), "id")

	index := New()
	assert.Nil(t, index.Build(root))
	assert.Equal(t, 1, index.Size())
}

func TestBuildRootFailureStaysFatal(t *testing.T) {
	// testing: a failure on the library root itself still aborts,
	// otherwise an empty index would mark every track as new
	root := filepath.Join(t.TempDir(), "missing")
	assert.Error(t, New().Build(root))
}

func TestBuildIgnoresCloseFailures(t *testing.T) {
	// testing: Build discards the outcome of closing each tag it opens,
	// so nothing about a close can abort the indexing
	root := t.TempDir()
	writeTrackFile(t, filepath.Join(root, "fname.mp3"), "")

	assert.Nil(t, New().Build(root))
}

func TestGetFallsBackToPath(t *testing.T) {
	index := New()
	index.SetPath("Artist - Title.mp3", Offline)

	status, ok := index.Get(&entity.Track{ID: "different-id", Title: "Title", Artists: []string{"Artist"}})
	assert.True(t, ok)
	assert.Equal(t, Offline, status)
}

func TestGetPrefersIDOverPath(t *testing.T) {
	index := New()
	index.SetPath("Artist - Title.mp3", Offline)
	index.SetID("id", Flush)

	status, ok := index.Get(&entity.Track{ID: "id", Title: "Title", Artists: []string{"Artist"}})
	assert.True(t, ok)
	assert.Equal(t, Flush, status)
}

func TestGetID(t *testing.T) {
	index := New()
	index.SetID("id", Online)

	status, ok := index.GetID("id")
	assert.True(t, ok)
	assert.Equal(t, Online, status)

	_, ok = index.GetID("missing")
	assert.False(t, ok)
}

func TestGetPath(t *testing.T) {
	index := New()
	index.SetPath("Artist - Title.mp3", Offline)

	status, ok := index.GetPath("Artist - Title.mp3")
	assert.True(t, ok)
	assert.Equal(t, Offline, status)

	_, ok = index.GetPath("Other - Track.mp3")
	assert.False(t, ok)
}

func TestGetIDAndPathDistinguishCollision(t *testing.T) {
	index := New()
	index.Set(&entity.Track{ID: "id-a", Title: "Title", Artists: []string{"Artist"}}, Offline)

	_, idKnown := index.GetID("id-a")
	assert.True(t, idKnown)

	// same filename, different track: id miss, path hit
	_, idKnown = index.GetID("id-b")
	assert.False(t, idKnown)
	_, pathKnown := index.GetPath((&entity.Track{ID: "id-b", Title: "Title", Artists: []string{"Artist"}}).Path().Final())
	assert.True(t, pathKnown)
}

func TestBuildCaseInsensitiveExtension(t *testing.T) {
	// testing: an uppercase extension is indexed like its lowercase twin
	root := t.TempDir()
	writeTrackFile(t, filepath.Join(root, "Artist - Title.MP3"), "id")

	index := New()
	assert.Nil(t, index.Build(root))
	assert.Equal(t, 1, index.Size())
}
