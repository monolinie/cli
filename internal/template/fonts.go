package template

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed fonts/*.woff2
var fontFiles embed.FS

// writeFonts copies the embedded font files into src/app/fonts/.
func writeFonts(dir string) error {
	fontsDir := filepath.Join(dir, "src", "app", "fonts")
	if err := os.MkdirAll(fontsDir, 0755); err != nil {
		return err
	}

	entries, err := fontFiles.ReadDir("fonts")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		data, err := fontFiles.ReadFile("fonts/" + entry.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(fontsDir, entry.Name()), data, 0644); err != nil {
			return err
		}
	}

	return nil
}
