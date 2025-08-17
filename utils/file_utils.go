package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func IsMP3File(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".mp3"
}

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func FormatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func FindMP3Files(dir string, recursive bool) ([]string, error) {
	var files []string

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && !recursive && path != dir {
			return filepath.SkipDir
		}

		if !info.IsDir() && IsMP3File(path) {
			files = append(files, path)
		}

		return nil
	}

	err := filepath.Walk(dir, walkFunc)
	return files, err
}

func CreateBackup(filePath string) (string, error) {
	backupPath := filePath + ".backup"

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	err = os.WriteFile(backupPath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

func RestoreBackup(filePath string) error {
	backupPath := filePath + ".backup"

	if !FileExists(backupPath) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	return nil
}

func GetFileName(filePath string) string {
	return filepath.Base(filePath)
}

func GetDirectory(filePath string) string {
	return filepath.Dir(filePath)
}

func CleanFileName(fileName string) string {
	invalid := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	result := fileName

	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	return result
}

func EnsureDirectoryExists(dir string) error {
	if !IsDirectory(dir) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	return nil
}

func GetRelativePath(base, target string) (string, error) {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}
	return rel, nil
}

func IsValidImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	validExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"}

	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}

	return false
}

func GetFileExtension(filePath string) string {
	return strings.ToLower(filepath.Ext(filePath))
}

func ChangeFileExtension(filePath, newExt string) string {
	dir := filepath.Dir(filePath)
	name := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	return filepath.Join(dir, name+newExt)
}

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
