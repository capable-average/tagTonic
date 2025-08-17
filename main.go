package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"tagTonic/cmd"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	if err := cmd.Execute(); err != nil {
		logrus.Fatal(err)
		os.Exit(1)
	}
}
