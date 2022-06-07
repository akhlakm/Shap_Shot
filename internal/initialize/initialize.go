package initialize

import (
	"fmt"
	"snap/internal/argparser"
	"snap/internal/fileutils"
	"snap/internal/logger"
	"snap/internal/settings"
)

func Execute() {
	logger.Trace("init-execute", "")
	args := argparser.GetParser()

	errmsg := "\nPlease run 'init' with a rootname (a name for the current project directory),\n" +
		"and a path to a remote folder to backup to.\n" +
		"\nUSAGE: init <rootname> <remote folder path>\n"

	rootname := args.ReqStr(1, errmsg)
	remotepath := args.ReqStr(2, errmsg)

	remotepath = fileutils.PathNormalize(remotepath)
	remotepath, _ = fileutils.AbsolutePath(remotepath)
	cwd, _ := fileutils.AbsolutePath(fileutils.CurrentWD())
	// sanity check
	if remotepath == cwd {
		logger.Error("init-execute", remotepath, "Cannot set current directory as a remote.")
	}

	if fileutils.IsASubPath(cwd, remotepath) {
		logger.Error("init-execute", remotepath, "Cannot set a remote inside the current directory.")
	}

	settings.Create(rootname, remotepath)
	settings.Write()

	msg := fmt.Sprintf("OK -- current directory initialized as a project root.\n"+
		"RootName:\t%s\nRemotePath:\t%s\n"+
		"\nYou can now use one of the following commands, all of them are safe to run.\n"+
		"pull, shot, list\n"+
		"\nYou can rerun 'init' if you want to use a different RootName or RemotePath.", rootname, remotepath)
	logger.Print(msg)

	logger.Done("init-execute", "")
}
