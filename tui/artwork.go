package tui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"tagTonic/mp3"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
)

type ArtworkRenderer struct {
	cache     *Cache
	tagEditor mp3.TagEditor
}

type ArtworkResult struct {
	Content   string
	IsKitty   bool
	Error     error
	ImageData []byte
}

type ArtworkRenderMsg struct {
	FilePath string
	Result   ArtworkResult
}

func NewArtworkRenderer(cache *Cache) *ArtworkRenderer {
	return &ArtworkRenderer{
		cache:     cache,
		tagEditor: mp3.NewTagEditor(),
	}
}

func (ar *ArtworkRenderer) RenderArtwork(filePath string) ArtworkResult {
	//try cache first
	if data, exists := ar.cache.GetArtwork(filePath); exists {
		return ar.renderArtworkData(data)
	}

	tags, err := ar.tagEditor.ReadTags(filePath)
	if err != nil {
		return ArtworkResult{
			Content: "Error reading file",
			Error:   err,
		}
	}

	if len(tags.Artwork) == 0 {
		return ArtworkResult{
			Content: "No artwork found",
		}
	}

	ar.cache.SetArtwork(filePath, tags.Artwork)

	return ar.renderArtworkData(tags.Artwork)
}

func (ar *ArtworkRenderer) RenderArtworkDataAsync(filePath string, data []byte) tea.Cmd {
	return func() tea.Msg {
		result := make(chan ArtworkResult, 1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					result <- ArtworkResult{
						Content: fmt.Sprintf("Artwork rendering panicked: %v", r),
						Error:   fmt.Errorf("panic: %v", r),
					}
				}
			}()

			rendered := ar.renderArtworkData(data)
			result <- rendered
		}()

		select {
		case artworkResult := <-result:
			return ArtworkRenderMsg{
				FilePath: filePath,
				Result:   artworkResult,
			}
		case <-time.After(2 * time.Second):
			size := len(data)
			var sizeStr string
			if size > 1024*1024 {
				sizeStr = fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
			} else if size > 1024 {
				sizeStr = fmt.Sprintf("%.1fKB", float64(size)/1024)
			} else {
				sizeStr = fmt.Sprintf("%dB", size)
			}

			return ArtworkRenderMsg{
				FilePath: filePath,
				Result: ArtworkResult{
					Content:   fmt.Sprintf("Artwork: %s (rendering timed out)", sizeStr),
					ImageData: data,
					Error:     fmt.Errorf("rendering timed out"),
				},
			}
		}
	}
}

func (ar *ArtworkRenderer) renderArtworkData(data []byte) ArtworkResult {
	if len(data) == 0 {
		return ArtworkResult{
			Content: "No artwork embedded",
		}
	}

	size := len(data)
	var sizeStr string
	if size > 1024*1024 {
		sizeStr = fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
	} else if size > 1024 {
		sizeStr = fmt.Sprintf("%.1fKB", float64(size)/1024)
	} else {
		sizeStr = fmt.Sprintf("%dB", size)
	}

	width, height, format, err := ar.GetImageInfo(data)
	var infoStr string
	if err == nil {
		infoStr = fmt.Sprintf("%dx%d %s", width, height, format)
	} else {
		infoStr = "unknown format"
	}

	if isKittySupported() {
		result := ar.renderWithKittyIcat(data, 0, 0)
		if result.Error != nil {
			return ArtworkResult{
				Content:   fmt.Sprintf("Artwork: %s %s (render failed: %v)", infoStr, sizeStr, result.Error),
				ImageData: data,
				Error:     result.Error,
			}
		}
		return result
	} else {
		return ArtworkResult{
			Content:   fmt.Sprintf("Artwork: %s %s (Kitty protocol not supported)", infoStr, sizeStr),
			ImageData: data,
		}
	}
}

func (ar *ArtworkRenderer) renderWithKittyIcat(data []byte, width, height int) ArtworkResult {
	return ar.renderWithKittyIcatAt(data, width, height, 0, 0)
}

func (ar *ArtworkRenderer) renderWithKittyIcatAt(data []byte, width, height, xPos, yPos int) ArtworkResult {
	if err := clearKittyImages(); err != nil {
		logrus.Debugf("Failed to clear kitty images: %v", err)
	}

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("tagtonic-artwork-%d.img", time.Now().UnixNano()))

	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return ArtworkResult{
			Content:   "Failed to create temp file",
			ImageData: data,
			Error:     err,
		}
	}

	defer os.Remove(tmpFile)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if width == 0 {
		width = 40
	}
	if height == 0 {
		height = 20
	}

	var cmd *exec.Cmd

	if xPos > 0 || yPos > 0 {
		placeArg := fmt.Sprintf("--place=%dx%d@%dx%d", width, height, xPos, yPos)
		cmd = exec.CommandContext(ctx, "kitty", "+kitten", "icat",
			"--transfer-mode=file",
			"--scale-up",
			"--z-index=-1",
			placeArg,
			tmpFile,
		)
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
	} else {
		cmd = exec.CommandContext(ctx, "kitty", "+kitten", "icat",
			"--transfer-mode=file",
			"--scale-up",
			tmpFile)
	}

	tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return ArtworkResult{
			Content:   fmt.Sprintf("Failed to open /dev/tty: %v", err),
			ImageData: data,
			Error:     err,
		}
	}
	defer tty.Close()

	var stderr bytes.Buffer
	cmd.Stdout = tty
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMs := stderr.String()
		fmt.Printf("[Kitty Debug] cmd: %v\nstderr: %s\n", cmd.Args, errMs)
		fmt.Printf("[Kitty Debug] TERM=%s, TERM_PROGRAM=%s, KITTY_WINDOW_ID=%s\n",
			os.Getenv("TERM"), os.Getenv("TERM_PROGRAM"), os.Getenv("KITTY_WINDOW_ID"))

		if ctx.Err() == context.DeadlineExceeded {
			return ArtworkResult{
				Content:   "Artwork rendering timed out",
				ImageData: data,
				Error:     fmt.Errorf("kitty +kitten icat timed out"),
			}
		}

		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return ArtworkResult{
			Content:   fmt.Sprintf("kitty +kitten icat failed: %s", errMsg),
			ImageData: data,
			Error:     err,
		}
	}

	return ArtworkResult{
		Content:   "Artwork displayed",
		IsKitty:   true,
		ImageData: data,
	}
}

func clearKittyImages() error {
	tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w", err)
	}
	defer tty.Close()

	deleteCmd := "\x1b_Ga=d,d=A\x1b\\"

	_, err = tty.Write([]byte(deleteCmd))
	if err != nil {
		return fmt.Errorf("failed to write delete command: %w", err)
	}

	time.Sleep(100 * time.Millisecond)

	return nil
}

func isKittySupported() bool {
	if os.Getenv("TAGTONIC_DISABLE_KITTY") == "1" {
		return false
	}

	if os.Getenv("TAGTONIC_FORCE_KITTY") == "1" {
		return true
	}

	term := os.Getenv("TERM")
	if strings.Contains(term, "kitty") || term == "xterm-kitty" {
		return true
	}

	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram == "kitty" || termProgram == "WezTerm" {
		return true
	}

	return os.Getenv("KITTY_WINDOW_ID") != "" || os.Getenv("KITTY_PID") != ""
}

func (ar *ArtworkRenderer) RenderArtworkWithSizeAndPosition(data []byte, width, height, xPos, yPos int) ArtworkResult {
	if len(data) == 0 {
		return ArtworkResult{
			Content: "No artwork embedded",
		}
	}

	if !isKittySupported() {
		size := len(data)
		var sizeStr string
		if size > 1024*1024 {
			sizeStr = fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
		} else if size > 1024 {
			sizeStr = fmt.Sprintf("%.1fKB", float64(size)/1024)
		} else {
			sizeStr = fmt.Sprintf("%dB", size)
		}

		return ArtworkResult{
			Content:   fmt.Sprintf("Artwork: %s (Kitty protocol not supported)", sizeStr),
			ImageData: data,
		}
	}

	return ar.renderWithKittyIcatAt(data, width, height, xPos, yPos)
}

func (ar *ArtworkRenderer) RenderArtworkWithSizeAndPositionAsync(filePath string, data []byte, width, height, xPos, yPos int) tea.Cmd {
	return func() tea.Msg {
		result := make(chan ArtworkResult, 1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					result <- ArtworkResult{
						Content: fmt.Sprintf("Artwork rendering panicked: %v", r),
						Error:   fmt.Errorf("panic: %v", r),
					}
				}
			}()

			rendered := ar.RenderArtworkWithSizeAndPosition(data, width, height, xPos, yPos)
			result <- rendered
		}()

		select {
		case artworkResult := <-result:
			return ArtworkRenderMsg{
				FilePath: filePath,
				Result:   artworkResult,
			}
		case <-time.After(2 * time.Second):
			size := len(data)
			var sizeStr string
			if size > 1024*1024 {
				sizeStr = fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
			} else if size > 1024 {
				sizeStr = fmt.Sprintf("%.1fKB", float64(size)/1024)
			} else {
				sizeStr = fmt.Sprintf("%dB", size)
			}

			return ArtworkRenderMsg{
				FilePath: filePath,
				Result: ArtworkResult{
					Content:   fmt.Sprintf("Artwork: %s (rendering timed out)", sizeStr),
					ImageData: data,
					Error:     fmt.Errorf("rendering timed out"),
				},
			}
		}
	}
}

func (ar *ArtworkRenderer) GetImageInfo(data []byte) (width, height int, format string, err error) {
	if len(data) == 0 {
		return 0, 0, "", fmt.Errorf("no image data")
	}

	reader := bytes.NewReader(data)
	img, format, err := image.Decode(reader)
	if err != nil {
		return 0, 0, "", err
	}

	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy(), format, nil
}

func CreateNoArtworkPlaceholder() string {
	return "No artwork embedded - Press 'ctrl+a' to fetch"
}
