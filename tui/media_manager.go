package tui

import (
	"tagTonic/fetcher"

	tea "github.com/charmbracelet/bubbletea"
)

type MediaManager struct {
	artworkRenderer *ArtworkRenderer
	lyricsPanel     *LyricsPanel
	artworkFetcher  fetcher.ArtworkFetcher
	currentFile     string
	artworkResult   ArtworkResult
}

func NewMediaManager(cache *Cache) *MediaManager {
	return &MediaManager{
		artworkRenderer: NewArtworkRenderer(cache),
		lyricsPanel:     NewLyricsPanel(),
		artworkFetcher:  fetcher.NewArtworkFetcher(),
	}
}

func (mm *MediaManager) LoadFileAsyncWithPosition(filePath string, lyrics string, artwork []byte, width, height, xPos, yPos int) tea.Cmd {
	mm.currentFile = filePath

	clearKittyImages()

	mm.lyricsPanel.SetLyrics(lyrics)

	if len(artwork) > 0 {
		mm.artworkResult = ArtworkResult{
			Content: "Rendering artwork...",
		}
		return mm.artworkRenderer.RenderArtworkWithSizeAndPositionAsync(filePath, artwork, width, height, xPos, yPos)
	}

	mm.artworkResult = ArtworkResult{
		Content: "No artwork embedded",
	}
	return nil
}

func (mm *MediaManager) UpdateArtworkResult(result ArtworkResult) {
	mm.artworkResult = result
}

func (mm *MediaManager) GetArtworkResult() ArtworkResult {
	return mm.artworkResult
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

func (mm *MediaManager) RenderArtworkWithSizeAndPosition(width, height, xPos, yPos int) tea.Cmd {
	if mm.currentFile == "" || len(mm.artworkResult.ImageData) == 0 {
		return nil
	}

	return mm.artworkRenderer.RenderArtworkWithSizeAndPositionAsync(
		mm.currentFile,
		mm.artworkResult.ImageData,
		width,
		height,
		xPos,
		yPos,
	)
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

func (mm *MediaManager) HandleWindowResizeWithPosition(width, height, xPos, yPos int) tea.Cmd {
	if len(mm.artworkResult.ImageData) > 0 && mm.artworkResult.IsKitty {
		return mm.RenderArtworkWithSizeAndPosition(width, height, xPos, yPos)
	}
	return nil
}

func (mm *MediaManager) HasArtwork() bool {
	return len(mm.artworkResult.ImageData) > 0
}
