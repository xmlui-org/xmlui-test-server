package localsvr

import (
	"os"
	"path/filepath"
	"strings"
)

func HomeRelative(d string) string {
	var homeDir string
	homeDir, err := os.UserHomeDir()
	if err != nil {
		goto end
	}
	if !strings.HasPrefix(d, homeDir) {
		goto end
	}
	d, _ = filepath.Rel(homeDir, d)
	d = "~/" + d
end:
	return d
}
