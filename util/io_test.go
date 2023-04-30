package util

import (
	"errors"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestFileMove(t *testing.T) {
	src, dst := "/tmp/test_a.txt", "/tmp/test_b.txt"
	file, err := os.Create(src)
	assert.Nil(t, err)
	assert.Nil(t, file.Close())
	assert.Nil(t, FileMoveOrCopy(src, dst))
	assert.Nil(t, os.Remove(dst))
}

func TestFileCopy(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.Rename, func() error {
			return errors.New("not renaming")
		}).
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return []byte{}, nil
		}).
		ApplyFunc(os.WriteFile, func() error {
			return nil
		}).
		ApplyFunc(os.Remove, func() error {
			return nil
		}).
		Reset()

	// testing
	assert.Nil(t, FileMoveOrCopy("/a", "/a"))
}

func TestFileCopyReadFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.Rename, func() error {
			return errors.New("not renaming")
		}).
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return nil, errors.New("ko")
		}).
		Reset()

	// testing
	assert.Error(t, FileMoveOrCopy("/a", "/a"), "ko")
}

func TestFileCopyWriteFailure(t *testing.T) {
	// monkey patching
	defer gomonkey.NewPatches().
		ApplyFunc(os.Rename, func() error {
			return errors.New("not renaming")
		}).
		ApplyFunc(os.ReadFile, func() ([]byte, error) {
			return []byte{}, nil
		}).
		ApplyFunc(os.WriteFile, func() error {
			return errors.New("ko")
		}).
		Reset()

	// testing
	assert.Error(t, FileMoveOrCopy("/a", "/a"), "ko")
}

func TestFileBaseStem(t *testing.T) {
	assert.Equal(t, "hello", FileBaseStem("hello.txt"))
}
