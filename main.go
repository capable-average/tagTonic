package main

import (
	"tagTonic/cmd"

	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	if err := cmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
