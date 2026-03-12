package cmd

import (
	"errors"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/streambinder/id3v2-sylt"
	"github.com/streambinder/spotitube/downloader"
	"github.com/streambinder/spotitube/entity"
	"github.com/streambinder/spotitube/lyrics"
	"github.com/streambinder/spotitube/processor"
	"github.com/streambinder/spotitube/spotify"
	"github.com/streambinder/spotitube/sys"
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
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).Return(id3v2.NewEmptyTag(), nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).Return(_track, nil).Build()
	mockey.Mock(lyrics.Search).Return("", nil).Build()
	mockey.Mock(downloader.Download).To(func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
		ch[0] <- []byte{}
		return nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&id3v2.Tag{}, "Save")).Return(nil).Build()

	// testing
	assert.Nil(t, sys.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")))
}

func TestCmdAttachOpenFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")), "ko")
}

func TestCmdAttachAuthFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).Return(id3v2.NewEmptyTag(), nil).Build()
	mockey.Mock(spotify.Authenticate).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")), "ko")
}

func TestCmdAttachTrackFailure(t *testing.T) {
	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).Return(id3v2.NewEmptyTag(), nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).Return(nil, errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")), "ko")
}

func TestCmdAttachLyricsFailure(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdAttachLyricsFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).Return(id3v2.NewEmptyTag(), nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).Return(_track, nil).Build()
	mockey.Mock(lyrics.Search).Return("", errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")), "ko")
}

func TestCmdAttacDownloadFailure(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdAttacDownloadFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).Return(id3v2.NewEmptyTag(), nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).Return(_track, nil).Build()
	mockey.Mock(lyrics.Search).Return("", nil).Build()
	mockey.Mock(downloader.Download).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")), "ko")
}

func TestCmdAttachSaveFailure(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdAttachSaveFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).Return(id3v2.NewEmptyTag(), nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).Return(_track, nil).Build()
	mockey.Mock(lyrics.Search).Return("", nil).Build()
	mockey.Mock(downloader.Download).To(func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
		ch[0] <- []byte{}
		return nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&id3v2.Tag{}, "Save")).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAttach(), "/path", "spotifyid")), "ko")
}

func TestCmdAttachRenameFailure(t *testing.T) {
	_track := &entity.Track{ID: "TestCmdAttachRenameFailure", Title: "Title", Artists: []string{"Artist"}}

	// monkey patching
	defer mockey.UnPatchAll()
	mockey.Mock(id3v2.Open).Return(id3v2.NewEmptyTag(), nil).Build()
	mockey.Mock(spotify.Authenticate).Return(&spotify.Client{}, nil).Build()
	mockey.Mock(mockey.GetMethod(&spotify.Client{}, "Track")).Return(_track, nil).Build()
	mockey.Mock(lyrics.Search).Return("", nil).Build()
	mockey.Mock(downloader.Download).To(func(_, _ string, _ processor.Processor, ch ...chan []byte) error {
		ch[0] <- []byte{}
		return nil
	}).Build()
	mockey.Mock(mockey.GetMethod(&id3v2.Tag{}, "Save")).Return(nil).Build()
	mockey.Mock(sys.FileMoveOrCopy).Return(errors.New("ko")).Build()

	// testing
	assert.EqualError(t, sys.ErrOnly(testExecute(cmdAttach(), "--rename", "/path", "spotifyid")), "ko")
}
