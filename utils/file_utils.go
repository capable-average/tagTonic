package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func DeriveTitleFromFilename(filePath string) string {
	base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	base = regexp.MustCompile(`(?i)^[0-9]{1,3}[-_. ]+`).ReplaceAllString(base, "")
	base = strings.NewReplacer("_", " ", "-", " ").Replace(base)
	fields := strings.Fields(base)
	if len(fields) == 0 {
		return base
	}
	return strings.Join(fields, " ")
}

func ValidateMP3File(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	if filepath.Ext(filePath) != ".mp3" {
		return fmt.Errorf("file is not an MP3: %s", filePath)
	}

	return nil
}
