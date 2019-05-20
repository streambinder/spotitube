package spotify

import (
	spttb_system "system"

	api "github.com/zmb3/spotify"
)

var (
	clientChannel       = make(chan *api.Client)
	clientState         = spttb_system.RandString(20)
	clientAuthenticator api.Authenticator
)
