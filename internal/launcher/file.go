package launcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ActionSpec is the payload carried by a File Action item. The Alfred
// workflow passes it verbatim to `lingbox launcher action`.
type ActionSpec struct {
	Action string `json:"action"`
	Path   string `json:"path"`
	Format string `json:"format,omitempty"`
}

// FileItems lists the ling-box actions that apply to the selected files.
func FileItems(paths []string) []Item {
	var items []Item
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			items = append(items, Item{
				Title:    "Cannot read " + filepath.Base(path),
				Subtitle: err.Error(),
				Valid:    false,
			})
			continue
		}
		if info.IsDir() {
			continue
		}

		base := filepath.Base(path)
		switch strings.ToLower(filepath.Ext(path)) {
		case ".dbf":
			items = append(items,
				fileActionItem("dbf-info", "DBF: Show info", base, path, ""),
				fileActionItem("dbf-export", "DBF: Export JSON", base, path, "json"),
				fileActionItem("dbf-export", "DBF: Export CSV", base, path, "csv"),
			)
		case ".json", ".yaml", ".yml", ".csv", ".md", ".markdown":
			source := sourceFormat(path)
			for _, target := range []string{"json", "yaml", "csv", "markdown"} {
				if target == source {
					continue
				}
				items = append(items, fileActionItem("convert", "Convert to "+target, base, path, target))
			}
		case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tif", ".tiff":
			items = append(items, fileActionItem("imgcat", "View in Terminal", base, path, ""))
		}
	}
	if len(items) == 0 {
		items = append(items, Item{
			Title:    "No ling-box action for this selection",
			Subtitle: "Supported: .dbf, .json, .yaml, .csv, .md and images",
			Valid:    false,
		})
	}
	return items
}

func fileActionItem(action, title, base, path, format string) Item {
	spec, _ := json.Marshal(ActionSpec{Action: action, Path: path, Format: format})
	return Item{
		UID:      fmt.Sprintf("%s:%s:%s", action, format, path),
		Title:    title,
		Subtitle: base,
		Arg:      string(spec),
		Valid:    true,
		Text:     &Text{Largetype: path},
	}
}

func sourceFormat(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".csv":
		return "csv"
	case ".md", ".markdown":
		return "markdown"
	}
	return ""
}
