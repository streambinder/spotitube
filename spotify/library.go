package spotify

import (
	"context"
	"errors"

	"github.com/zmb3/spotify/v2"
)

func (client *Client) Library(limit int, channels ...chan interface{}) error {
	var (
		ctx          = context.Background()
		library, err = client.CurrentUsersTracks(ctx)
	)
	if err != nil {
		return err
	}

	ctr := 0
	for {
		for _, libraryTrack := range library.Tracks {
			track := trackEntity(libraryTrack.FullTrack)
			for _, ch := range channels {
				ch <- track
			}
			if ctr++; limit > 0 && ctr >= limit {
				return nil
			}
		}

		if err := client.NextPage(ctx, library); errors.Is(err, spotify.ErrNoMorePages) {
			break
		} else if err != nil {
			return err
		}
	}

	return nil
}
