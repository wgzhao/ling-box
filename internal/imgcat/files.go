package imgcat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// imageExtensions is the set of recognized image file extensions, lower-case.
var imageExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".bmp":  true,
	".webp": true,
	".tiff": true,
}

// IsImageFile reports whether name has a supported image file extension.
// The check is case-insensitive ("photo.PNG" is accepted).
func IsImageFile(name string) bool {
	return imageExtensions[strings.ToLower(filepath.Ext(name))]
}

// ListImages returns the paths of all image files directly inside dir,
// sorted by filename. Subdirectories are not traversed.
func ListImages(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var images []string
	for _, e := range entries {
		if e.IsDir() || !IsImageFile(e.Name()) {
			continue
		}
		images = append(images, filepath.Join(dir, e.Name()))
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("no image files found in %s", dir)
	}
	return images, nil
}
