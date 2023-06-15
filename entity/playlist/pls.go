package playlist

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gosimple/slug"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/util"
)

type PLSEncoder struct {
	target  string
	data    []byte
	entries int
}

func (encoder *PLSEncoder) init(name string) error {
	encoder.target = slug.Make(name) + ".pls"
	encoder.data = []byte(fmt.Sprintf("[%s]\n\n", name))
	encoder.entries = 0
	return nil
}

func (encoder *PLSEncoder) Add(track *entity.Track) error {
	encoder.entries += 1
	encoder.data = append(encoder.data, []byte(
		fmt.Sprintf("File%d=%s\nTitle%d=%s\nLength%d=%d\n\n",
			encoder.entries,
			filepath.Base(track.Path().Final()),
			encoder.entries,
			util.FileBaseStem(filepath.Base(track.Path().Final())),
			encoder.entries,
			track.Duration,
		),
	)...)
	return nil
}

func (encoder *PLSEncoder) Close() error {
	encoder.data = append(encoder.data, []byte(
		fmt.Sprintf("NumberOfEntries=%d\n", encoder.entries),
	)...)
	return os.WriteFile(encoder.target, encoder.data, 0o644)
}
