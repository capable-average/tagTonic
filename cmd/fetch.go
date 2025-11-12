package cmd

import (
	"path/filepath"
	"strings"

	"tagTonic/config"
	"tagTonic/fetcher"
	"tagTonic/mp3"
	"tagTonic/utils"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	fetchLyrics       bool
	fetchArtwork      bool
	force             bool
	clearLyricsFetch  bool
	clearArtworkFetch bool
)

var fetchCmd = &cobra.Command{
	Use:   "fetch [file]",
	Short: "Fetch lyrics and artwork from APIs",
	Long: `Fetch lyrics and artwork from various APIs and embed them into MP3 files.
	
Examples:
  tagTonic fetch song.mp3 --lyrics --artwork
  tagTonic fetch song.mp3 --lyrics
  tagTonic fetch song.mp3 --artwork --force`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]

		if err := utils.ValidateMP3File(filePath); err != nil {
			logrus.Fatal(err)
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			logrus.Debugf("Failed to load config, using defaults: %v", err)
			cfg = config.DefaultConfig()
		}

		editor := mp3.NewTagEditor()
		
		var lyricsFetcher fetcher.LyricsFetcher
		if cfg.GeniusAPIKey != "" {
			lyricsFetcher = fetcher.NewLyricsFetcherWithConfig(cfg.GeniusAPIKey)
			logrus.Debug("Using Genius API with authentication")
		} else {
			lyricsFetcher = fetcher.NewLyricsFetcher()
			logrus.Debug("Using Genius API without authentication (limited access)")
		}
		
		artworkFetcher := fetcher.NewArtworkFetcher()

		tags, err := editor.ReadTags(filePath)
		if err != nil {
			logrus.Fatalf("Failed to read current tags: %v", err)
		}

		updates := mp3.TagUpdates{ClearLyrics: clearLyricsFetch, ClearArtwork: clearArtworkFetch}

		sourceTitle := tags.Title
		if strings.TrimSpace(sourceTitle) == "" {
			sourceTitle = utils.DeriveTitleFromFilename(filePath)
			logrus.Debugf("Derived title from filename: %s", sourceTitle)
		}
		sourceArtist := tags.Artist

		if fetchLyrics {
			if tags.Lyrics != "" && !force {
				logrus.Info("Lyrics already present (use --force to overwrite)")
			} else {
				logrus.Info("Fetching lyrics...")
				lyrics, err := lyricsFetcher.Fetch(sourceTitle, sourceArtist)
				if err != nil {
					logrus.Warnf("Failed to fetch lyrics: %v", err)
				} else {
					updates.Lyrics = lyrics
					logrus.Info("Lyrics fetched successfully")
				}
			}
		}

		if fetchArtwork {
			if len(tags.Artwork) > 0 && !force {
				logrus.Info("Artwork already present (use --force to overwrite)")
			} else {
				logrus.Info("Fetching artwork...")
				artwork, err := artworkFetcher.Fetch(sourceTitle, sourceArtist, tags.Album)
				if err != nil {
					logrus.Warnf("Failed to fetch artwork: %v", err)
				} else {
					updates.Artwork = artwork
					logrus.Info("Artwork fetched successfully")
				}
			}
		}

		if updates.Lyrics != "" || updates.Artwork != nil || updates.ClearLyrics || updates.ClearArtwork {
			if err := editor.EditTags(filePath, updates); err != nil {
				logrus.Fatalf("Failed to update tags: %v", err)
			}
			logrus.Infof("Successfully updated %s", filepath.Base(filePath))
		} else {
			logrus.Info("No new content to update")
		}
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

	fetchCmd.Flags().BoolVar(&fetchLyrics, "lyrics", false, "Fetch lyrics")
	fetchCmd.Flags().BoolVar(&fetchArtwork, "artwork", false, "Fetch artwork")
	fetchCmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing content")
	fetchCmd.Flags().BoolVar(&clearLyricsFetch, "clear-lyrics", false, "Remove existing lyrics (no fetch)")
	fetchCmd.Flags().BoolVar(&clearArtworkFetch, "clear-artwork", false, "Remove existing artwork (no fetch)")
}
