package config

import (
	"fmt"
	"os/user"
	"path/filepath"
)

const (
	// HomePath represents the identifier for home dir
	HomePath = iota
	// CachePath represents the identifier for cache dir
	CachePath
	_
)

var (
	cacheDir   = filepath.Join(".cache", "spotitube")
	configPath = filepath.Join(".config", "spotitube")
)

// RelativeTo returns a path string extending
// the given path type
func RelativeTo(path string, pathType int) string {
	switch pathType {
	case CachePath:
		return filepath.Join(
			RelativeTo(filepath.Join(cacheDir, path), HomePath),
		)
	case HomePath:
		usr, _ := user.Current()
		return filepath.Join(usr.HomeDir, path)
	}

	return ""
}

// CacheDir returns cache dir
func CacheDir() string {
	return RelativeTo(cacheDir, HomePath)
}

// Path returns config path
func Path() string {
	return RelativeTo(configPath, HomePath)
}

// CacheBin returns a cached binary dir
// by its given version
func CacheBin(version int) string {
	return fmt.Sprintf(RelativeTo("spotitube.%d", CachePath), version)
}
