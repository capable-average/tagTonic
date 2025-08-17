package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"tagTonic/fetcher"
	"tagTonic/mp3"
	"tagTonic/utils"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	batchDir       string
	batchPattern   string
	batchLyrics    bool
	batchArtwork   bool
	batchRecursive bool
	batchForce     bool
)

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Batch process MP3 files",
	Long: `Process multiple MP3 files in a directory.
	
Examples:
  tagTonic batch --dir ./music/ --lyrics --artwork
  tagTonic batch --dir ./music/ --pattern "*.mp3" --lyrics
  tagTonic batch --dir ./music/ --recursive --artwork`,
	Run: func(cmd *cobra.Command, args []string) {
		if !batchLyrics && !batchArtwork {
			logrus.Info("No action flags provided (use --lyrics and/or --artwork)")
			return
		}
		if batchDir == "" {
			logrus.Fatal("Directory is required. Use --dir flag")
		}

		if _, err := os.Stat(batchDir); os.IsNotExist(err) {
			logrus.Fatalf("Directory does not exist: %s", batchDir)
		}

		files, err := findMP3Files(batchDir, batchPattern, batchRecursive)
		if err != nil {
			logrus.Fatalf("Failed to find MP3 files: %v", err)
		}

		if len(files) == 0 {
			logrus.Info("No MP3 files found")
			return
		}

		logrus.Infof("Found %d MP3 files to process", len(files))

		editor := mp3.NewTagEditor()
		lyricsFetcher := fetcher.NewLyricsFetcher()
		artworkFetcher := fetcher.NewArtworkFetcher()

		processedCount := 0
		updatedCount := 0
		skippedCount := 0
		errorCount := 0

		for _, file := range files {
			processedCount++
			logrus.Infof("Processing: %s", filepath.Base(file))

			tags, err := editor.ReadTags(file)
			if err != nil {
				logrus.Errorf("Failed to read tags for %s: %v", filepath.Base(file), err)
				errorCount++
				continue
			}

			updates := mp3.TagUpdates{}

			sourceTitle := tags.Title
			if strings.TrimSpace(sourceTitle) == "" {
				sourceTitle = utils.DeriveTitleFromFilename(file)
				logrus.Debugf("Derived title from filename (%s): %s", filepath.Base(file), sourceTitle)
			}
			sourceArtist := tags.Artist

			if batchLyrics {
				if tags.Lyrics != "" && !batchForce {
					logrus.Debugf("Lyrics already present for %s (skip; --force to overwrite)", filepath.Base(file))
				} else {
					lyrics, err := lyricsFetcher.Fetch(sourceTitle, sourceArtist)
					if err != nil {
						logrus.Warnf("Failed to fetch lyrics for %s: %v", filepath.Base(file), err)
					} else {
						updates.Lyrics = lyrics
					}
				}
			}

			if batchArtwork {
				if len(tags.Artwork) > 0 && !batchForce {
					logrus.Debugf("Artwork already present for %s (skip; --force to overwrite)", filepath.Base(file))
				} else {
					artwork, err := artworkFetcher.Fetch(sourceTitle, sourceArtist, tags.Album)
					if err != nil {
						logrus.Warnf("Failed to fetch artwork for %s: %v", filepath.Base(file), err)
					} else {
						updates.Artwork = artwork
					}
				}
			}

			if updates.Lyrics != "" || updates.Artwork != nil {
				if err := editor.EditTags(file, updates); err != nil {
					logrus.Errorf("Failed to update tags for %s: %v", filepath.Base(file), err)
					errorCount++
					continue
				}
				updatedCount++
			} else {
				skippedCount++
			}
		}

		logrus.Infof("Batch complete: processed=%d updated=%d skipped=%d errors=%d", processedCount, updatedCount, skippedCount, errorCount)
	},
}

func init() {
	rootCmd.AddCommand(batchCmd)

	batchCmd.Flags().StringVar(&batchDir, "dir", "", "Directory containing MP3 files")
	batchCmd.Flags().StringVar(&batchPattern, "pattern", "*.mp3", "File pattern to match")
	batchCmd.Flags().BoolVar(&batchLyrics, "lyrics", false, "Fetch lyrics for all files")
	batchCmd.Flags().BoolVar(&batchArtwork, "artwork", false, "Fetch artwork for all files")
	batchCmd.Flags().BoolVar(&batchRecursive, "recursive", false, "Search recursively in subdirectories")
	batchCmd.Flags().BoolVar(&batchForce, "force", false, "Force overwrite existing lyrics or artwork")

	batchCmd.MarkFlagRequired("dir")
}

func findMP3Files(dir, pattern string, recursive bool) ([]string, error) {
	var files []string

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && !recursive && path != dir {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			matched, err := filepath.Match(pattern, info.Name())
			if err != nil {
				return err
			}
			if matched {
				files = append(files, path)
			}
		}

		return nil
	}

	err := filepath.Walk(dir, walkFunc)
	return files, err
}
