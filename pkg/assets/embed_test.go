package assets

import (
	"io/fs"
	"strings"
	"testing"
)

func TestAllTemplatesEmbedded(t *testing.T) {
	var files []string
	fs.WalkDir(Static, "static/templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Fatal(err)
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	// Verify that files starting with '_' are embedded (requires the all: prefix).
	for _, want := range []string{"_footer.html", "_header.html", "_navigation.html", "index.html"} {
		found := false
		for _, f := range files {
			if strings.HasSuffix(f, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing embedded template file: %s", want)
		}
	}
}
