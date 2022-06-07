package status

import (
	"fmt"
	"io/ioutil"
	"snap/internal/argparser"
	"snap/internal/fileutils"
	"snap/internal/history"
	"snap/internal/logger"
	"snap/internal/settings"
	"sort"
)

func Execute() {
	args := argparser.GetParser()
	remote := settings.DefaultRemote()
	if !fileutils.DirExists(remote) {
		errmsg := "Remote directory does not exist.\n" +
			"\nMake sure it is mounted. Or take your first snapshot and it will be created automatically.\n"
		logger.Error("show-history", remote, errmsg)
	}

	rootname := settings.RootName()

	ssid, err := args.GetInt(1)
	if err != nil {
		snaplist := list_snap_files(remote, rootname)
		for _, snap := range snaplist {
			show_snap_info(remote, rootname, snap)
		}
	} else {
		hist := history.Make(ssid, remote, rootname)
		if !hist.SnapFileExists() {
			logger.Error("show-history", fmt.Sprint(ssid), "No such snapshot exists in the remote.")
		}
		hist.Load()
		logger.Print("\nCommitted Changes:\n")
		hist.Print()
	}

	currSS := settings.LastSnapshot()
	logger.Print(fmt.Sprintf("Last snapshot synced: %d", currSS))
	logger.Print("\nPlease specify a snapshot number to see a list of file changes.")
	logger.Print("Or run 'shot' to see a list of current changes from the last snapshot.")
}

func show_snap_info(remote string, rootname string, ssname string) {
	snap := history.Make(0, remote, rootname)
	if snap.SnapFileOfNameExists(ssname) {
		snap.LoadFileMeta(ssname)

		logger.Print(fmt.Sprintf(
			"%s\n       %s      [%s]\n", snap.GetMeta("DATE"), ssname, snap.GetMeta("CRUD")))
		// logger.Print(fmt.Sprintf("       %s\n", snap.GetMeta("DESC")))
	} else {
		logger.Trace("snaplist-show", fmt.Sprintf("no such snapshot: %s ", ssname))
	}
}

func list_snap_files(remote, rootname string) []string {
	histDir := fileutils.SSHistoryDir(remote, rootname)
	files, err := ioutil.ReadDir(histDir)
	if err != nil {
		if !fileutils.DirExists(histDir) {
			errmsg := "Remote history does not exist.\n" +
				"\nMake sure the remote is mounted. Or take your first snapshot and it will be created automatically.\n"
			logger.Error("show-history", histDir, errmsg)
		}
	}

	var snapnames []string
	for _, f := range files {
		if !f.IsDir() {
			snapnames = append(snapnames, f.Name())
		}
	}
	sort.Strings(snapnames)
	return snapnames
}
