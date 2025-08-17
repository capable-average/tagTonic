package cmd

import (
	"fmt"
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "tagTonic",
	Short: "A powerful MP3 tag editor with TUI interface",
	Long: `tagTonic is a modern CLI and TUI application for editing MP3 tags.
	
Features:
- Manual MP3 tag editing (title, artist, album, lyrics, artwork)
- Automatic lyrics and artwork fetching from APIs
- Batch processing of MP3 files
- Beautiful TUI interface with animations
- Image preview and engaging UI with lipgloss

Examples:
  tagTonic edit song.mp3
  tagTonic fetch --lyrics --artwork song.mp3
  tagTonic batch --dir ./music/
  tagTonic tui`,
	Version: "1.0.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tagTonic.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".tagTonic")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}

	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}
