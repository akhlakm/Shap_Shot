package initialize

import (
	"snap/internal/argparser"
	"snap/internal/fileutils"
	"snap/internal/logger"
	"snap/internal/settings"
)

func Execute() {
	logger.Trace("init-execute", "")
	args := argparser.GetParser()

	rootname := args.ReqStr(1, "USAGE: init <rootname> <remotepath>")
	remotepath := args.ReqStr(2, "USAGE: init <rootname> <remotepath>")

	remotepath, _ = fileutils.AbsolutePath(remotepath)
	cwd, _ := fileutils.AbsolutePath(fileutils.CurrentWD())
	// sanity check
	if remotepath == cwd {
		logger.Error("init-execute", remotepath, "Cannot set current directory as remote.")
	}

	settings.Create(rootname, remotepath)
	settings.Write()
	logger.Done("init-execute", "")
}
