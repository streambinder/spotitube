package spotify

import (
	"context"
	"errors"
	"fmt"

	"github.com/streambinder/spotitube/sys"
	"github.com/zmb3/spotify/v2"
)

func (client *Client) Random(searchType spotify.SearchType, amount int, channels ...chan interface{}) error {
	var (
		ctx         = context.Background()
		search, err = client.Search(context.Background(), fmt.Sprintf("%c*", sys.RandomAlpha()), searchType, spotify.Limit(amount))
	)
	if err != nil {
		return err
	}

	for {
		for _, fullTrack := range search.Tracks.Tracks {
			track := trackEntity(fullTrack)
			for _, ch := range channels {
				ch <- track
			}
		}

		if err := client.NextPage(ctx, search.Tracks); errors.Is(err, spotify.ErrNoMorePages) {
			break
		} else if err != nil {
			return err
		}
	}

	return nil
}
