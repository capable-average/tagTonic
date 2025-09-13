package cmd

import (
	"tagTonic/tui"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive terminal UI",
	Run: func(cmd *cobra.Command, args []string) {
		if err := tui.Run(); err != nil {
			logrus.Fatalf("TUI exited with error: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
