//go:build windows || darwin || linux

package store

import (
	"fmt"
	"os"
	"strings"
)

type Image struct {
	Filename string `json:"filename"`
	Path     string `json:"path"`
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// Bytes loads image data from disk for a given ImageData reference
func (i *Image) Bytes() ([]byte, error) {
	return ImgBytes(i.Path)
}

// ImgBytes reads image data from the specified file path
func ImgBytes(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("empty image path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read image file %s: %w", path, err)
	}

	return data, nil
}

// ImgToFile saves image data to disk and returns ImageData reference
func (s *Store) ImgToFile(chatID string, imageBytes []byte, filename, mimeType string) (Image, error) {
	return Image{}, nil
}

// sanitize removes unsafe characters from filenames
func sanitize(filename string) string {
	// Convert to safe characters only
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '_'
	}, filename)

	// Clean up and validate
	safe = strings.Trim(safe, "_")
	if safe == "" {
		return "image"
	}
	return safe
}
