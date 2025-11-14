package tui

import (
	"tagTonic/fetcher"

	tea "github.com/charmbracelet/bubbletea"
)

type MediaManager struct {
	lyricsPanel     *LyricsPanel
	artworkFetcher  fetcher.ArtworkFetcher
	currentFile     string
}

func NewMediaManager() *MediaManager {
	return &MediaManager{
		lyricsPanel:     NewLyricsPanel(),
		artworkFetcher:  fetcher.NewArtworkFetcher(),
	}
}

func (mm *MediaManager) LoadFileAsyncWithPosition(filePath string, lyrics string, artwork []byte, width, height, xPos, yPos int) tea.Cmd {
	mm.currentFile = filePath
	mm.lyricsPanel.SetLyrics(lyrics)
	return nil
}

func (mm *MediaManager) GetLyricsPanel() *LyricsPanel {
	return mm.lyricsPanel
}

func (mm *MediaManager) FetchArtwork(title, artist, album string) tea.Cmd {
	if mm.currentFile == "" {
		return nil
	}

	return func() tea.Msg {
		artwork, err := mm.artworkFetcher.Fetch(title, artist, album)
		return ArtworkFetchedMsg{
			Artwork: artwork,
			Error:   err,
		}
	}
}

func (mm *MediaManager) FetchLyrics(title, artist string) tea.Cmd {
	return func() tea.Msg {
		lyrics, err := mm.lyricsPanel.fetcher.Fetch(title, artist)
		return LyricsFetchedMsg{
			Lyrics: lyrics,
			Error:  err,
		}
	}
}
