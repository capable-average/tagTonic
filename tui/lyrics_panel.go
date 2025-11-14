package tui

import (
	"strings"
	"tagTonic/config"
	"tagTonic/fetcher"
)

type LyricsPanel struct {
	lyrics         string
	originalLyrics string
	editBuffer     string
	isEditing      bool
	isDirty        bool
	scrollOffset   int
	lines          []string
	fetcher        fetcher.LyricsFetcher
	isLoading      bool
	fetchError     string
}

func NewLyricsPanel() *LyricsPanel {
	cfg, _ := config.LoadConfig()
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	var lyricsFetcher fetcher.LyricsFetcher
	if cfg.GeniusAPIKey != "" {
		lyricsFetcher = fetcher.NewLyricsFetcherWithConfig(cfg.GeniusAPIKey)
	} else {
		lyricsFetcher = fetcher.NewLyricsFetcher()
	}

	return &LyricsPanel{
		fetcher: lyricsFetcher,
		lines:   make([]string, 0),
	}
}

func (lp *LyricsPanel) SetLyrics(lyrics string) {
	lp.lyrics = lyrics
	lp.originalLyrics = lyrics
	lp.scrollOffset = 0
	lp.isDirty = false
}

func (lp *LyricsPanel) GetLyrics() string {
	if lp.isEditing {
		return lp.editBuffer
	}
	return lp.lyrics
}

func (lp *LyricsPanel) StartEditing() {
	lp.isEditing = true
	lp.editBuffer = lp.lyrics
}

func (lp *LyricsPanel) StopEditing() {
	if !lp.isEditing {
		return
	}

	lp.lyrics = lp.editBuffer
	lp.isEditing = false
	lp.isDirty = lp.lyrics != lp.originalLyrics
}

func (lp *LyricsPanel) CancelEditing() {
	lp.isEditing = false
	lp.editBuffer = ""
}

func (lp *LyricsPanel) UpdateEditBuffer(content string) {
	lp.editBuffer = content
}

func (lp *LyricsPanel) GetEditBuffer() string {
	return lp.editBuffer
}

func (lp *LyricsPanel) IsEditing() bool {
	return lp.isEditing
}

func (lp *LyricsPanel) IsDirty() bool {
	return lp.isDirty
}

func (lp *LyricsPanel) PageUp(pageSize int) {
	lp.scrollOffset -= pageSize
	if lp.scrollOffset < 0 {
		lp.scrollOffset = 0
	}
}

func (lp *LyricsPanel) PageDown(pageSize int) {
	maxOffset := len(lp.lines) - pageSize
	if maxOffset < 0 {
		maxOffset = 0
	}

	lp.scrollOffset += pageSize
	if lp.scrollOffset > maxOffset {
		lp.scrollOffset = maxOffset
	}
}

func (lp *LyricsPanel) GetVisibleLines(height int) []string {
	if len(lp.lines) == 0 {
		return []string{"No lyrics available"}
	}

	start := lp.scrollOffset
	end := start + height

	if start >= len(lp.lines) {
		return []string{}
	}

	if end > len(lp.lines) {
		end = len(lp.lines)
	}

	return lp.lines[start:end]
}

func (lp *LyricsPanel) GetTotalLines() int {
	return len(lp.lines)
}

func (lp *LyricsPanel) FetchLyrics(title, artist string) {
	lp.isLoading = true
	lp.fetchError = ""

	go func() {
		lyrics, err := lp.fetcher.Fetch(title, artist)

		lp.isLoading = false

		if err != nil {
			lp.fetchError = err.Error()
		} else {
			lp.SetLyrics(lyrics)
		}
	}()
}

func (lp *LyricsPanel) GetFetchError() string {
	return lp.fetchError
}

func (lp *LyricsPanel) ClearFetchError() {
	lp.fetchError = ""
}

func (lp *LyricsPanel) HasLyrics() bool {
	return strings.TrimSpace(lp.lyrics) != ""
}
