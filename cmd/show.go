package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"tagTonic/mp3"
	"tagTonic/tui"
	"tagTonic/utils"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var showJSON bool
var showArtwork bool

var showCmd = &cobra.Command{
	Use:   "show [file]",
	Short: "Show current tags for an MP3 file",
	Long:  "Reads and prints the existing tag metadata (title, artist, album, genre, year, lyrics length, artwork size).",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file := args[0]
		if err := utils.ValidateMP3File(file); err != nil {
			logrus.Fatal(err)
		}
		editor := mp3.NewTagEditor()
		tags, err := editor.ReadTags(file)
		if err != nil {
			logrus.Fatalf("failed to read tags: %v", err)
		}
		derivedTitle := ""
		if strings.TrimSpace(tags.Title) == "" {
			derivedTitle = utils.DeriveTitleFromFilename(file)
		}
		if showJSON {
			out := struct {
				File         string `json:"file"`
				*mp3.MP3Tags `json:"tags"`
				LyricsLength int    `json:"lyricsLength"`
				ArtworkBytes int    `json:"artworkBytes"`
				DerivedTitle string `json:"derivedTitle,omitempty"`
			}{File: filepath.Base(file), MP3Tags: tags, LyricsLength: len(tags.Lyrics), ArtworkBytes: len(tags.Artwork), DerivedTitle: derivedTitle}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(out); err != nil {
				logrus.Fatal(err)
			}
			return
		}
		tw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
		fmt.Fprintf(tw, "File:\t%s\n", filepath.Base(file))
		fmt.Fprintf(tw, "Title:\t%s\n", tags.Title)
		if derivedTitle != "" {
			fmt.Fprintf(tw, "(Derived Title):\t%s\n", derivedTitle)
		}
		fmt.Fprintf(tw, "Artist:\t%s\n", tags.Artist)
		fmt.Fprintf(tw, "Album:\t%s\n", tags.Album)
		fmt.Fprintf(tw, "Genre:\t%s\n", tags.Genre)
		fmt.Fprintf(tw, "Year:\t%d\n", tags.Year)
		fmt.Fprintf(tw, "Lyrics Length:\t%d chars\n", len(tags.Lyrics))
		fmt.Fprintf(tw, "Artwork Size:\t%d bytes\n", len(tags.Artwork))
		tw.Flush()

		if showArtwork {
			fmt.Println()
			cache := tui.NewCache(10)
			artworkRenderer := tui.NewArtworkRenderer(cache)
			result := artworkRenderer.RenderArtwork(file)

			if result.Error != nil {
				logrus.Warnf("Failed to display artwork: %v", result.Error)
			} else {
				fmt.Println(result.Content)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")
	showCmd.Flags().BoolVar(&showArtwork, "artwork", false, "Display artwork in terminal (if supported)")
}
