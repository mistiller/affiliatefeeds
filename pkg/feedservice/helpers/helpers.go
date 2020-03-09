package helpers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FindFolderDir returns a directory path for the specified folder to anchor realtive paths to
// In testing the current working directory could be different from execution;
// in these cases it helps to find the top level dir of the app rathe than the tested package
func FindFolderDir(name string) string {
	wd, _ := os.Getwd()
	for !strings.HasSuffix(wd, name) {
		wd = filepath.Dir(wd)
	}
	return wd
}

// IsOnline sends get request to https://www.google.com and returns true if no error
func IsOnline(url string) bool {
	if len(url) == 0 {
		url = "https://www.google.com/"
	}
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return true
}

func getMIME(incipit []byte) string {
	var magicTable = map[string]string{
		"\xff\xd8\xff":      "image/jpeg",
		"\x89PNG\r\n\x1a\n": "image/png",
		"GIF87a":            "image/gif",
		"GIF89a":            "image/gif",
	}

	incipitStr := string(incipit)
	for magic, mime := range magicTable {
		if strings.HasPrefix(incipitStr, magic) {
			return mime
		}
	}

	return ""
}
