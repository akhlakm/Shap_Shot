package check

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"snap/internal/argparser"
	"snap/internal/fileutils"
	"snap/internal/history"
	"snap/internal/logger"
	"snap/internal/settings"
	"strings"
)

func Execute() {
	args := argparser.GetParser()

	remote := settings.DefaultRemote()
	if !fileutils.DirExists(remote) {
		errmsg := "Remote directory does not exist.\n" +
			"\nMake sure it is mounted. Or take your first snapshot and it will be created automatically.\n"
		logger.Error("restore-execute", remote, errmsg)
	}

	rootname := settings.RootName()

	errmsg := "\nUSAGE: check <path to file/dir to checkout> [<snapshot id>]\n"

	checkoutPath := args.ReqStr(1, errmsg)
	remotePath := fileutils.PathJoin(fileutils.BackPath(remote, rootname), checkoutPath)

	ssid, err := args.GetInt(2)
	if err != nil {
		ssid = 0
	}

	if ssid > 0 {
		hist := history.Make(ssid, remote, rootname)
		if !hist.SnapFileExists() {
			logger.Error("check-ssid", fmt.Sprint(ssid), "No such snapshot exists in the remote.")
		}
	}

	if !fileutils.DirExists(remotePath) {
		errmsg := "No such file/directory exists in the remote.\n" +
			"\nPlease run list [<snapshot id>] for a complete list of available files."

		logger.Error("check-path", remotePath, errmsg)
	}

	ncopy := copy_directory(remotePath, ssid)
	logger.Print(fmt.Sprintf("%d files copied", ncopy))
}

func copy_directory(remotePath string, ssid int) int {
	ccount := 0
	filepath.WalkDir(remotePath, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			logger.Error("check-copy-path", remotePath, "Failed to walk remote directory.")
			return e
		}

		// copy the files
		if !d.IsDir() {

			// specific version specified
			if ssid > 0 {
				ssname := fileutils.FormatSnap(ssid)
				if !strings.HasPrefix(d.Name(), ssname) {
					// do not copy
					return nil
				}
			}

			relpath, err := fileutils.CalcRelativePath(remotePath, s)
			if err != nil {
				logger.Error("check-copy-path", s, "Failed to determine relative path.")
			}

			dstpath := fileutils.ShotPath(relpath)

			//@todo: check bytes copied.
			cpbytes, err := fileutils.CopyFile(s, dstpath)
			if err != nil {
				fmt.Println(err)
				logger.Error("copy-file", s, "Failed to copy file.")
			}
			ccount++
			logger.Print(fmt.Sprintf("OK -- %s (%d bytes)", relpath, cpbytes))
		}
		return nil
	})

	return ccount
}
