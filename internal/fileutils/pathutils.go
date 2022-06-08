package fileutils

import (
	"path/filepath"
)

func PathMatch(pattern, path string) bool {
	// filepath.Match("/home/catch/*", "/home/catch/foo") == true
	// filepath.Match("/home/catch/*", "/home/catch/foo/bar") == false
	// filepath.Match("/home/?opher", "/home/gopher") == true
	// filepath.Match("/home/\\*", "/home/*") == true

	res, err := filepath.Match(pattern, path)

	if err == nil {
		return res
	} else {
		return false
	}
}
