package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FileBrowser struct {
	currentDir      string
	entries         []FileEntry
	filteredEntries []FileEntry
	selectedIndex   int
	selectedFiles   map[string]bool
	showHidden      bool
	batchMode       bool
}

type FileEntry struct {
	Name  string
	Path  string
	IsDir bool
	Size  int64
	IsMP3 bool
}

func NewFileBrowser(startDir string) *FileBrowser {
	absDir, err := filepath.Abs(startDir)
	if err != nil || absDir == "" {
		absDir = startDir
	}
	absDir = filepath.Clean(absDir)

	fb := &FileBrowser{
		currentDir:      absDir,
		entries:         make([]FileEntry, 0),
	}
	fb.LoadDirectory()
	return fb
}

func (fb *FileBrowser) LoadDirectory() error {
	items, err := os.ReadDir(fb.currentDir)
	if err != nil {
		return err
	}

	fb.entries = make([]FileEntry, 0)

	parentDir := filepath.Dir(fb.currentDir)
	if fb.currentDir != "/" && parentDir != fb.currentDir {
		fb.entries = append(fb.entries, FileEntry{
			Name:  "..",
			Path:  parentDir,
			IsDir: true,
		})
	}

	for _, item := range items {
		if !fb.showHidden && strings.HasPrefix(item.Name(), ".") {
			continue
		}

		fullPath := filepath.Join(fb.currentDir, item.Name())
		info, err := item.Info()
		if err != nil {
			continue
		}

		entry := FileEntry{
			Name:  item.Name(),
			Path:  fullPath,
			IsDir: item.IsDir(),
			Size:  info.Size(),
			IsMP3: strings.HasSuffix(strings.ToLower(item.Name()), ".mp3"),
		}

		if entry.IsDir || entry.IsMP3 {
			fb.entries = append(fb.entries, entry)
		}
	}

	sort.Slice(fb.entries, func(i, j int) bool {
		if fb.entries[i].IsDir != fb.entries[j].IsDir {
			return fb.entries[i].IsDir
		}
		return strings.ToLower(fb.entries[i].Name) < strings.ToLower(fb.entries[j].Name)
	})

	return nil
}

func (fb *FileBrowser) Navigate() error {

	selected := fb.entries[fb.selectedIndex]
	if selected.IsDir {
		fb.currentDir = selected.Path
		fb.selectedIndex = 0
		return fb.LoadDirectory()
	}

	return nil
}

func (fb *FileBrowser) GetSelectedFile() *FileEntry {
	if fb.selectedIndex >= len(fb.entries) || len(fb.entries) == 0 {
		return nil
	}

	selected := &fb.entries[fb.selectedIndex]
	if selected.IsMP3 {
		return selected
	}

	return nil
}

func (fb *FileBrowser) MoveUp() {
	if fb.selectedIndex > 0 {
		fb.selectedIndex--
	}
}

func (fb *FileBrowser) MoveDown() {
	if fb.selectedIndex < len(fb.entries)-1 {
		fb.selectedIndex++
	}
}

func (fb *FileBrowser) PageUp(pageSize int) {
	fb.selectedIndex -= pageSize
	if fb.selectedIndex < 0 {
		fb.selectedIndex = 0
	}
}

func (fb *FileBrowser) PageDown(pageSize int) {
	fb.selectedIndex += pageSize
	if fb.selectedIndex >= len(fb.entries) {
		fb.selectedIndex = len(fb.entries) - 1
	}
	if fb.selectedIndex < 0 {
		fb.selectedIndex = 0
	}
}

func (fb *FileBrowser) GetCurrentDir() string {
	return fb.currentDir
}

func (fb *FileBrowser) GetFilteredEntries() []FileEntry {
	return fb.filteredEntries
}

func (fb *FileBrowser) GetSelectedIndex() int {
	return fb.selectedIndex
}

func (fb *FileBrowser) ToggleHidden() {
	fb.showHidden = !fb.showHidden
	fb.LoadDirectory()
}

func (fb *FileBrowser) IsSelected(path string) bool {
	return fb.selectedFiles[path]
}

func (fb *FileBrowser) IsBatchMode() bool {
	return fb.batchMode
}