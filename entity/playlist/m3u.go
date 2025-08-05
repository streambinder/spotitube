package playlist

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gosimple/slug"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/sys"
)

type M3UEncoder struct {
	target string
	data   []byte
}

func (encoder *M3UEncoder) init(name string) error {
	encoder.target = slug.Make(name) + ".m3u"
	encoder.data = []byte(
		fmt.Sprintf("#EXTM3U\n#PLAYLIST:%s\n", name),
	)

	return nil
}

func (encoder *M3UEncoder) Add(track *entity.Track) error {
	encoder.data = append(encoder.data, []byte(
		fmt.Sprintf(
			"#EXTINF:%s,%s\n%s\n",
			strconv.Itoa(track.Duration),
			sys.FileBaseStem(filepath.Base(track.Path().Final())),
			filepath.Base(track.Path().Final()),
		),
	)...)
	return nil
}

func (encoder *M3UEncoder) Close() error {
	return os.WriteFile(encoder.target, encoder.data, 0o600)
}
