package cmd

import (
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/bogem/id3v2/v2"
	"github.com/streambinder/spotitube/downloader"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/processor"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/util"
	"github.com/stretchr/testify/assert"
)

func BenchmarkAttach(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TestCmdAttach(&testing.T{})
	}
}

func TestCmdAttach(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdAttach", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
			return id3v2.NewEmptyTag(), nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return _track, nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "", nil
		}).
		ApplyFunc(downloader.Download, func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
			ch[0] <- []byte{}
			return nil
		}).
		ApplyMethod(&id3v2.Tag{}, "Save", func() error {
			return nil
		}).
		Reset()

	// testing
	assert.Nil(t, util.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")))
}

func TestCmdAttachOpenFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
		return nil, errors.New("ko")
	}).Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")), "ko")
}

func TestCmdAttachAuthFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
			return id3v2.NewEmptyTag(), nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")), "ko")
}

func TestCmdAttachTrackFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
			return id3v2.NewEmptyTag(), nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")), "ko")
}

func TestCmdAttachLyricsFailure(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdAttachLyricsFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
			return id3v2.NewEmptyTag(), nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return _track, nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "", errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")), "ko")
}

func TestCmdAttacDownloadFailure(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdAttacDownloadFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
			return id3v2.NewEmptyTag(), nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return _track, nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "", nil
		}).
		ApplyFunc(downloader.Download, func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")), "ko")
}

func TestCmdAttachSaveFailure(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdAttachSaveFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
			return id3v2.NewEmptyTag(), nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return _track, nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "", nil
		}).
		ApplyFunc(downloader.Download, func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
			ch[0] <- []byte{}
			return nil
		}).
		ApplyMethod(&id3v2.Tag{}, "Save", func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")), "ko")
}

func TestCmdAttachRenameFailure(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdAttachRenameFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(id3v2.Open, func() (*id3v2.Tag, error) {
			return id3v2.NewEmptyTag(), nil
		}).
		ApplyFunc(spotify.Authenticate, func() (*spotify.Client, error) {
			return &spotify.Client{}, nil
		}).
		ApplyMethod(&spotify.Client{}, "Track", func() (*entity.Track, error) {
			return _track, nil
		}).
		ApplyFunc(lyrics.Search, func() (string, error) {
			return "", nil
		}).
		ApplyFunc(downloader.Download, func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
			ch[0] <- []byte{}
			return nil
		}).
		ApplyMethod(&id3v2.Tag{}, "Save", func() error {
			return nil
		}).
		ApplyFunc(util.FileMoveOrCopy, func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.EqualError(t, util.ErrOnly(testExecute(cmdAttach(), "--rename", "/path", "spotifyid")), "ko")
}
