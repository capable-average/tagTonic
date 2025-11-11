package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tagTonic/config"
	"tagTonic/fetcher"
	"tagTonic/mp3"
	"tagTonic/utils"

	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	batchDir         string
	batchPattern     string
	batchLyrics      bool
	batchArtwork     bool
	batchRecursive   bool
	batchForce       bool
	batchWorkers     int
	batchNoProgress  bool
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

		logrus.Infof("Found %d MP3 files to process with %d workers", len(files), batchWorkers)

		var bar *progressbar.ProgressBar
		if !batchNoProgress {
			bar = progressbar.NewOptions(len(files),
				progressbar.OptionSetDescription("Processing MP3 files"),
				progressbar.OptionSetWidth(50),
				progressbar.OptionShowCount(),
				progressbar.OptionShowIts(),
				progressbar.OptionSetItsString("files"),
				progressbar.OptionThrottle(100),
				progressbar.OptionShowElapsedTimeOnFinish(),
				progressbar.OptionSetRenderBlankState(true),
			)
		}

		stats := processBatchConcurrently(files, batchWorkers, bar)
		
		if bar != nil {
			bar.Finish()
		}
		
		logrus.Infof("Batch complete: processed=%d updated=%d skipped=%d errors=%d", 
			stats.processed, stats.updated, stats.skipped, stats.errors)
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
	batchCmd.Flags().IntVar(&batchWorkers, "workers", 5, "Number of concurrent workers (1-20)")
	batchCmd.Flags().BoolVar(&batchNoProgress, "no-progress", false, "Disable progress bar and show detailed logs")

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

type BatchStats struct {
	processed int
	updated   int
	skipped   int
	errors    int
	mu        sync.Mutex
}

func (s *BatchStats) incrementProcessed() {
	s.mu.Lock()
	s.processed++
	s.mu.Unlock()
}

func (s *BatchStats) incrementUpdated() {
	s.mu.Lock()
	s.updated++
	s.mu.Unlock()
}

func (s *BatchStats) incrementSkipped() {
	s.mu.Lock()
	s.skipped++
	s.mu.Unlock()
}

func (s *BatchStats) incrementErrors() {
	s.mu.Lock()
	s.errors++
	s.mu.Unlock()
}

type FileJob struct {
	filepath string
}

type FileResult struct {
	filepath string
	success  bool
	updated  bool
	error    error
}

func processBatchConcurrently(files []string, numWorkers int, bar *progressbar.ProgressBar) *BatchStats {
	if numWorkers < 1 {
		numWorkers = 1
	}
	if numWorkers > 20 {
		numWorkers = 20
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Debugf("Failed to load config, using defaults: %v", err)
		cfg = config.DefaultConfig()
	}

	stats := &BatchStats{}
	jobs := make(chan FileJob, len(files))
	results := make(chan FileResult, len(files))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(ctx, jobs, results, &wg, cfg)
	}

	go func() {
		defer close(jobs)
		for _, file := range files {
			select {
			case jobs <- FileJob{filepath: file}:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		stats.incrementProcessed()
		
		if bar != nil {
			bar.Add(1)
		}
		
		if result.error != nil {
			stats.incrementErrors()
			if bar != nil {
				bar.Describe(filepath.Base(result.filepath) + " - ERROR")
			} else {
				logrus.Errorf("Failed to process %s: %v", filepath.Base(result.filepath), result.error)
			}
		} else if result.updated {
			stats.incrementUpdated()
			if bar != nil && logrus.GetLevel() >= logrus.DebugLevel {
				bar.Describe(filepath.Base(result.filepath) + " - UPDATED")
			} else if bar == nil {
				logrus.Infof("Updated: %s", filepath.Base(result.filepath))
			}
		} else {
			stats.incrementSkipped()
			if bar != nil && logrus.GetLevel() >= logrus.DebugLevel {
				bar.Describe(filepath.Base(result.filepath) + " - SKIPPED")
			} else if bar == nil {
				logrus.Debugf("Skipped: %s", filepath.Base(result.filepath))
			}
		}
	}

	return stats
}

func worker(ctx context.Context, jobs <-chan FileJob, results chan<- FileResult, wg *sync.WaitGroup, cfg *config.Config) {
	defer wg.Done()

	editor := mp3.NewTagEditor()
	
	var lyricsFetcher fetcher.LyricsFetcher
	if cfg.GeniusAPIKey != "" {
		lyricsFetcher = fetcher.NewLyricsFetcherWithConfig(cfg.GeniusAPIKey)
	} else {
		lyricsFetcher = fetcher.NewLyricsFetcher()
	}
	
	artworkFetcher := fetcher.NewArtworkFetcher()

	for job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
			result := processFile(job.filepath, editor, lyricsFetcher, artworkFetcher)
			results <- result
		}
	}
}

func processFile(filePath string, editor mp3.TagEditor, lyricsFetcher fetcher.LyricsFetcher, artworkFetcher fetcher.ArtworkFetcher) FileResult {
	result := FileResult{filepath: filePath}

	tags, err := editor.ReadTags(filePath)
	if err != nil {
		result.error = err
		return result
	}

	updates := mp3.TagUpdates{}

	sourceTitle := tags.Title
	if strings.TrimSpace(sourceTitle) == "" {
		sourceTitle = utils.DeriveTitleFromFilename(filePath)
		logrus.Debugf("Derived title from filename (%s): %s", filepath.Base(filePath), sourceTitle)
	}
	sourceArtist := tags.Artist

	var lyricsWg sync.WaitGroup
	var lyricsResult string
	var lyricsError error
	var artworkResult []byte
	var artworkError error

	if batchLyrics {
		if tags.Lyrics != "" && !batchForce {
			logrus.Debugf("Lyrics already present for %s (skip; --force to overwrite)", filepath.Base(filePath))
		} else {
			lyricsWg.Add(1)
			go func() {
				defer lyricsWg.Done()
				lyricsResult, lyricsError = lyricsFetcher.Fetch(sourceTitle, sourceArtist)
			}()
		}
	}

	if batchArtwork {
		if len(tags.Artwork) > 0 && !batchForce {
			logrus.Debugf("Artwork already present for %s (skip; --force to overwrite)", filepath.Base(filePath))
		} else {
			lyricsWg.Add(1)
			go func() {
				defer lyricsWg.Done()
				artworkResult, artworkError = artworkFetcher.Fetch(sourceTitle, sourceArtist, tags.Album)
			}()
		}
	}

	lyricsWg.Wait()

	if batchLyrics && lyricsError == nil && lyricsResult != "" {
		updates.Lyrics = lyricsResult
	} else if batchLyrics && lyricsError != nil {
		logrus.Debugf("Failed to fetch lyrics for %s: %v", filepath.Base(filePath), lyricsError)
	}

	if batchArtwork && artworkError == nil && artworkResult != nil {
		updates.Artwork = artworkResult
	} else if batchArtwork && artworkError != nil {
		logrus.Debugf("Failed to fetch artwork for %s: %v", filepath.Base(filePath), artworkError)
	}

	if updates.Lyrics != "" || updates.Artwork != nil {
		if err := editor.EditTags(filePath, updates); err != nil {
			result.error = err
			return result
		}
		result.updated = true
	}

	result.success = true
	return result
}
