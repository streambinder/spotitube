package spotify

import (
	"errors"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
	"github.com/zmb3/spotify/v2"
)

func BenchmarkUser(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestUser(&testing.T{})
	}
}

func TestUser(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "CurrentUser", func() (*spotify.PrivateUser, error) {
			return &spotify.PrivateUser{User: spotify.User{DisplayName: "User"}}, nil
		}).
		Reset()

	// testing
	username, err := testClient().Username()
	assert.Nil(t, err)
	assert.Equal(t, "User", username)
}

func TestUserFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(time.Sleep, func() {}).
		ApplyMethod(&spotify.Client{}, "CurrentUser", func() (*spotify.PrivateUser, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testClient().Username()), "ko")
}
