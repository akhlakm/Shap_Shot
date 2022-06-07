package main

import (
	"os"
	"snap/internal/argparser"
	"snap/internal/initialize"
	"snap/internal/logger"
	"snap/internal/restore"
	"snap/internal/settings"
	"snap/internal/snapshot"
	"snap/internal/status"
)

func main() {
	logger.Trace("main", "")

	// parse args
	args := os.Args
	argparser.Create(args[1:])
	if len(args) > 1 {
		cmd := args[1]
		if cmd == "init" {
			initialize.Execute()
		} else {
			settings.Load()
			if cmd == "pull" {
				restore.Execute()
			} else if cmd == "list" {
				status.Execute()
			} else if cmd == "shot" {
				snapshot.Execute()
			} else {
				logger.Error("main", cmd, "Unknown argument")
			}
		}
	}

	logger.Print("~")
}
