package cmd

import (
	"os"
	"path/filepath"

	"tagTonic/mp3"
	"tagTonic/utils"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	title        string
	artist       string
	album        string
	genre        string
	year         int
	lyrics       string
	artwork      string
	clearLyrics  bool
	clearArtwork bool
)

var editCmd = &cobra.Command{
	Use:   "edit [file]",
	Short: "Edit MP3 tags manually",
	Long: `Edit MP3 tags manually with specified values.
	
Examples:
  tagTonic edit song.mp3 --title "New Title" --artist "New Artist"
  tagTonic edit song.mp3 --lyrics "path/to/lyrics.txt" --artwork "path/to/artwork.jpg"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]

		if err := utils.ValidateMP3File(filePath); err != nil {
			logrus.Fatal(err)
		}

		editor := mp3.NewTagEditor()

		updates := mp3.TagUpdates{Title: title, Artist: artist, Album: album, Genre: genre, Year: year, ClearLyrics: clearLyrics, ClearArtwork: clearArtwork}

		if lyrics != "" {
			lyricsContent, err := os.ReadFile(lyrics)
			if err != nil {
				logrus.Fatalf("Failed to read lyrics file: %v", err)
			}
			updates.Lyrics = string(lyricsContent)
		}

		if artwork != "" {
			artworkData, err := os.ReadFile(artwork)
			if err != nil {
				logrus.Fatalf("Failed to read artwork file: %v", err)
			}
			updates.Artwork = artworkData
		}

		if err := editor.EditTags(filePath, updates); err != nil {
			logrus.Fatalf("Failed to edit tags: %v", err)
		}

		logrus.Infof("Successfully updated tags for %s", filepath.Base(filePath))
	},
}

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().StringVar(&title, "title", "", "Set title")
	editCmd.Flags().StringVar(&artist, "artist", "", "Set artist")
	editCmd.Flags().StringVar(&album, "album", "", "Set album")
	editCmd.Flags().StringVar(&genre, "genre", "", "Set genre")
	editCmd.Flags().IntVar(&year, "year", 0, "Set year")
	editCmd.Flags().StringVar(&lyrics, "lyrics", "", "Path to lyrics file")
	editCmd.Flags().StringVar(&artwork, "artwork", "", "Path to artwork file")
	editCmd.Flags().BoolVar(&clearLyrics, "clear-lyrics", false, "Remove existing embedded lyrics")
	editCmd.Flags().BoolVar(&clearArtwork, "clear-artwork", false, "Remove existing embedded artwork")
}
