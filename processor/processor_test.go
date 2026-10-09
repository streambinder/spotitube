package processor

import (
	"os"
	"testing"

	"github.com/streambinder/spotitube/entity"
	"github.com/stretchr/testify/assert"
)

var track = &entity.Track{
	ID:       "123",
	Title:    "Title",
	Artists:  []string{"Artist"},
	Album:    "Album",
	Artwork:  entity.Artwork{URL: "http://ima.ge"},
	Duration: 180,
	Number:   1,
	Year:     1970,
}

func BenchmarkProcessor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestProcessorDo(&testing.T{})
	}
}

func TestProcessorDo(t *testing.T) {
	download := seedDownload(t, []byte{})
	fakeFFmpeg(t, "-23.45", 0, 0)

	assert.Nil(t, Do(track))

	// the encoder ran last: the downloaded file now carries an ID3 tag
	content, err := os.ReadFile(download)
	assert.Nil(t, err)
	assert.Equal(t, "ID3", string(content[:3]))
}

func TestProcessorDoFailure(t *testing.T) {
	useTempCache(t)
	// a negligible loudness offset skips normalization entirely, then
	// the encoder fails on the missing downloaded file
	fakeFFmpeg(t, "-14.0", 0, 0)

	assert.ErrorContains(t, Do(track), "no such file or directory")
}

func TestProcessorDoLoudnessSkipped(t *testing.T) {
	seedDownload(t, []byte{})
	fakeFFmpeg(t, "-23.45", 1, 0)

	// a skipped normalization runs the rest of the chain and reports the skip
	assert.ErrorIs(t, Do(track), ErrLoudnessSkipped)
}

func TestProcessorDoLoudnessSkippedEncoderFailure(t *testing.T) {
	useTempCache(t)
	fakeFFmpeg(t, "-23.45", 1, 0)

	// the chain continues past a skipped normalization, a later failure still aborts
	err := Do(track)
	assert.ErrorContains(t, err, "no such file or directory")
	assert.NotErrorIs(t, err, ErrLoudnessSkipped)
}
