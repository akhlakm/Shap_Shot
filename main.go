package main

import (
	"os"
	"snap/internal/argparser"
	"snap/internal/check"
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
			} else if cmd == "check" {
				check.Execute()
			} else if cmd == "shot" {
				snapshot.Execute()
			} else {
				logger.Error("main", cmd,
					"Unknown argument.\n"+
						"Please use one of the init, pull, shot, list commands.")
			}
		}
	} else {
		if settings.Exists() {
			logger.Error("main", "",
				"No argument.\n"+
					"\nPlease specify one of the following commands, all of them are safe to run.\n"+
					"pull, shot, list")
		} else {
			logger.Error("main", "",
				"Not initialized as a project root.\n"+
					"\nPlease run 'init' with a rootname (a name for the current project directory),\n"+
					"and a path to a remote folder to backup to, or restore from.\n\n"+
					"Make sure your rootname is future-proof. Each remote can contain multiple roots.\n"+
					"Once you initialized, you will be able to take a snapshot of the current directory,\n"+
					"or view a list of snapshots if you have an existing remote and restore them.\n"+
					"\nUSAGE: init <rootname> <remote folder path>\n")
		}
	}

	logger.Print("~")
}
