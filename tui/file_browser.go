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
	searchQuery     string
	showHidden      bool
	selectedFiles   map[string]bool
	isSearchMode    bool
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
		filteredEntries: make([]FileEntry, 0),
		selectedFiles:   make(map[string]bool),
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

	fb.applyFilter()

	if fb.selectedIndex >= len(fb.filteredEntries) {
		fb.selectedIndex = 0
	}

	return nil
}

func (fb *FileBrowser) Navigate() error {
	if fb.selectedIndex >= len(fb.filteredEntries) {
		return nil
	}

	selected := fb.filteredEntries[fb.selectedIndex]
	if selected.IsDir {
		fb.currentDir = selected.Path
		fb.selectedIndex = 0
		return fb.LoadDirectory()
	}

	return nil
}

func (fb *FileBrowser) GetSelectedFile() *FileEntry {
	if fb.selectedIndex >= len(fb.filteredEntries) || len(fb.filteredEntries) == 0 {
		return nil
	}

	selected := &fb.filteredEntries[fb.selectedIndex]
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
	if fb.selectedIndex < len(fb.filteredEntries)-1 {
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
	if fb.selectedIndex >= len(fb.filteredEntries) {
		fb.selectedIndex = len(fb.filteredEntries) - 1
	}
	if fb.selectedIndex < 0 {
		fb.selectedIndex = 0
	}
}

func (fb *FileBrowser) SetSearch(query string) {
	fb.searchQuery = query
	fb.isSearchMode = query != ""
	fb.applyFilter()
	fb.selectedIndex = 0
}

func (fb *FileBrowser) applyFilter() {
	if fb.searchQuery == "" {
		fb.filteredEntries = fb.entries
		return
	}

	fb.filteredEntries = make([]FileEntry, 0)
	query := strings.ToLower(fb.searchQuery)

	for _, entry := range fb.entries {
		if strings.Contains(strings.ToLower(entry.Name), query) {
			fb.filteredEntries = append(fb.filteredEntries, entry)
		}
	}
}

func (fb *FileBrowser) ClearSearch() {
	fb.searchQuery = ""
	fb.isSearchMode = false
	fb.applyFilter()
	fb.selectedIndex = 0
}

func (fb *FileBrowser) GetCurrentDir() string {
	return fb.currentDir
}

func (fb *FileBrowser) GetEntries() []FileEntry {
	return fb.entries
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

func (fb *FileBrowser) ToggleSelection() {
	if fb.selectedIndex >= len(fb.filteredEntries) {
		return
	}

	selected := fb.filteredEntries[fb.selectedIndex]
	if !selected.IsMP3 {
		return
	}

	if fb.selectedFiles[selected.Path] {
		delete(fb.selectedFiles, selected.Path)
	} else {
		fb.selectedFiles[selected.Path] = true
	}
}

func (fb *FileBrowser) IsSelected(path string) bool {
	return fb.selectedFiles[path]
}

func (fb *FileBrowser) GetSelectedFiles() []string {
	var files []string
	for path := range fb.selectedFiles {
		files = append(files, path)
	}
	return files
}

func (fb *FileBrowser) ClearSelection() {
	fb.selectedFiles = make(map[string]bool)
}

func (fb *FileBrowser) ToggleBatchMode() {
	fb.batchMode = !fb.batchMode
	if !fb.batchMode {
		fb.ClearSelection()
	}
}

func (fb *FileBrowser) IsBatchMode() bool {
	return fb.batchMode
}

func (fb *FileBrowser) IsSearchMode() bool {
	return fb.isSearchMode
}

func (fb *FileBrowser) GetSearchQuery() string {
	return fb.searchQuery
}
