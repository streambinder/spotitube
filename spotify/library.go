package spotify

import (
	"errors"

	"github.com/zmb3/spotify"
)

func (client *Client) Library(channels ...chan interface{}) error {
	library, err := client.CurrentUsersTracks()
	if err != nil {
		return err
	}

	for {
		for _, libraryTrack := range library.Tracks {
			track := trackEntity(&libraryTrack.FullTrack)
			for _, ch := range channels {
				ch <- track
			}
		}

		if err := client.NextPage(library); errors.Is(err, spotify.ErrNoMorePages) {
			break
		} else if err != nil {
			return err
		}
	}

	return nil
}
