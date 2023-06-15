package spotify

import "context"

func (client *Client) Username() (string, error) {
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		return "", err
	}

	return user.DisplayName, nil
}
