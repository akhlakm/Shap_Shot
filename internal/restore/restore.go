package restore

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"snap/internal/argparser"
	"snap/internal/fileutils"
	"snap/internal/history"
	"snap/internal/logger"
	"snap/internal/settings"
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

	// load the last ss
	// can be 0 when new, or specific int
	lastss := settings.LastSnapshot()
	lastHistory := history.Make(lastss, remote, rootname)
	if lastss > 0 && !fileutils.SSExists(lastss, remote, rootname) {
		logger.Error("restore-load", fmt.Sprint(lastss),
			"Last snapshot does not exist in remote.\n"+
				"\nRoot settings suggest existence of previous snapshots.\n"+
				"Please rerun 'init' if needed, or run 'list' to see the available snapshots in the remote.")
	}

	lastHistory.Load()

	// status of current files in root
	localHistory := history.Make(0, remote, rootname)
	localHistory = walk_root(localHistory)
	check_local_modifications(lastHistory, localHistory)

	// calculate restore items
	newss, err := args.GetInt(1)
	if err != nil || newss < 1 {
		// no argument given
		newss = calc_latest_ssid(remote, rootname)
		if newss == 0 {
			logger.Print("No available snapshot to restore from remote.")
			return
		}
	}

	remoteHistory := history.Make(newss, remote, rootname)
	if newss > 0 && !fileutils.SSExists(newss, remote, rootname) {
		logger.Error("snapshot-load", fmt.Sprint(newss),
			"No such snapshot exists to restore.")
	}
	remoteHistory.Load()
	localHistory = calc_action_items(remoteHistory, localHistory)
	localHistory, _ = calculate_meta_items(localHistory)

	if args.HasFlag("--ignores") || args.HasFlag("-i") {
		localHistory.PrintCrud("I")
	}
	logger.Print("\nChanges to commit:\n")
	localHistory.Print()

	if args.HasFlag("--dry") || args.HasFlag("-n") {
		// --dry has a higher priority over --go
		logger.Print(fmt.Sprintf("\nDry run %d > %d. Snapshot is NOT restored.", lastss, newss))
		logger.Print("Please specify --go to commit the changes.")
	} else if args.HasFlag("--go") || args.HasFlag("-go") {
		perform_actions(localHistory)
		settings.SetLastSnapshot(remoteHistory.SnapId)
		settings.Write()
		logger.Print(fmt.Sprintf("Last snapshot synced: %d", remoteHistory.SnapId))
	} else {
		logger.Print(fmt.Sprintf("\nDry run %d > %d. Snapshot is NOT restored.", lastss, newss))
		logger.Print("Please specify --go to commit the changes.")
	}
}

func perform_actions(loc *history.Hist) {
	ccount := 0
	dcount := 0
	rootpath := fileutils.CurrentWD()
	for phash := range loc.RelPath {
		crud := loc.GetCrud(phash)
		relpath := loc.GetRelPath(phash)
		dstpath := fileutils.PathJoin(rootpath, relpath)
		// copy is create
		if crud == "C" || crud == "U" {
			srcpath := loc.GetRestorePath(phash)

			if !fileutils.FileExists(srcpath) {
				errmsg := "File does not exist in remote.\n" +
					"\nMake sure the files/ directory of the current root is okay\n" +
					"and the files are not missing. If you have manually deleted files\n" +
					"the file pointers in the shot files might be broken.\n" +
					"See a detail list of files first using the list <snapshot number> command.\n"

				logger.Error("restore-action", srcpath, errmsg)
			}
			logger.Trace("restore-copyfile", dstpath)
			cpbytes, err := fileutils.CopyFile(srcpath, dstpath)
			if err != nil {
				fmt.Println(err)
				logger.Error("restore-copyfile", srcpath, "Failed to copy file.")
			}
			ccount++
			// we know how many bytes should have been copied
			if !fileutils.FileSizeSame(loc.GetFileHash(phash), cpbytes) {
				logger.Print(fmt.Sprintf("WARNING -- %s (%d bytes) copy does not match with "+
					"the expected file size recorded in the remote.\n"+
					"\nIt can happen if remote file has been manually modified.\n"+
					"Please take a new snapshot if this is the case,\n"+
					"otherwise, make sure your remote files are in good conditions.\n", relpath, cpbytes))
			} else {
				logger.Print(fmt.Sprintf("OK -- %s (%d bytes)", relpath, cpbytes))
			}
		} else if crud == "D" {
			err := fileutils.DeleteFile(dstpath)
			if err != nil {
				errmsg := "Failed to delete file.\n" +
					"\nMake sure the file is not being accessed by another process.\n" +
					"Or try manually deleting it first.\n"
				logger.Error("restore-delete", dstpath, errmsg)
			} else {
				dcount++
				logger.Print(fmt.Sprintf("OK -- %s (delete)", relpath))
			}
		}
	}
	logger.Print(fmt.Sprintf("DONE -- %d files copied, %d files removed", ccount, dcount))
}

func calculate_meta_items(hist *history.Hist) (*history.Hist, []int) {
	create := hist.CountCrud("C")
	retain := hist.CountCrud("R")
	update := hist.CountCrud("U")
	delete := hist.CountCrud("D")
	ignore := hist.CountCrud("I")
	total := create + retain + update
	hist.SetMetaInt("FileCount", total)
	hist.SetMetaInt("IgnoreCount", ignore)

	// format crud: +9;=20;^2;-1
	crud := fmt.Sprintf("+%d;=%d;^%d;-%d", create, retain, update, delete)
	hist.SetMetaString("CRUD", crud)

	ncrud := []int{create, retain, update, delete}

	return hist, ncrud
}

func calc_action_items(rem, loc *history.Hist) *history.Hist {
	// For each path in the root,
	for _, phash := range loc.PathHashList() {
		if settings.ShouldIgnore(loc.GetRelPath(phash)) {
			loc.SetCrud(phash, "I")
		} else if !rem.IsPathHash(phash) {
			// no such file in the remote
			loc.SetCrud(phash, "D")
		}
	}

	// Foreach PathHash in remote,
	for _, phash := range rem.PathHashList() {
		// similar local file exists
		if loc.IsPathHash(phash) {
			if settings.ShouldIgnore(loc.GetRelPath(phash)) {
				loc.SetCrud(phash, "I")
			} else if rem.GetCrud(phash) == "D" {
				// the file was set to be deleted in the remote
				loc.SetCrud(phash, "D")
			} else {
				// 	if CU, Copy PathHash/02 to WD/Path/Name
				// 	if R, Copy PathHash/01 to WD/Path/Name
				remFHash := rem.GetFileHash(phash)
				locFHash := loc.GetFileHash(phash)
				if !fileutils.FileHashSame(remFHash, locFHash) {
					loc.SetCrud(phash, "U")
					// if we decided to update to the remote version
					loc.SetFileHash(phash, remFHash)
				} else {
					loc.SetCrud(phash, "R")
				}
				remTarget := rem.GetTarget(phash)
				loc.SetTarget(phash, remTarget)
			}
		} else {
			// copy everything else that hasn't been deleted in the remote
			if rem.GetCrud(phash) != "D" {
				loc.SetAction(phash, rem.GetAction(phash))
				remTarget := rem.GetTarget(phash)
				loc.SetTarget(phash, remTarget)
				if settings.ShouldIgnore(rem.GetRelPath(phash)) {
					loc.SetCrud(phash, "I")
				} else {
					loc.SetCrud(phash, "C")
				}
			}
		}
	}
	return loc
}

func check_local_modifications(last, curr *history.Hist) {
	if last.SnapId == 0 {
		return
	}
	for _, phash := range curr.PathHashList() {
		relpath := curr.GetRelPath(phash)
		if settings.ShouldIgnore(relpath) {
			continue
		}
		if !last.IsPathHash(phash) {
			// 	if PathHash not in LAST,
			// 	throw error, must shot first before restoring.
			logger.Error("restore-check-modifications", fmt.Sprintf("C %s", relpath),
				"Local modifications found, please take a snapshot first.")
		} else {
			// 	if PathHash in LAST,
			// 		compare_with_last_hash()
			// 		error if no match, must shot first, before restoring.
			oldFHash := last.GetFileHash(phash)
			newFHash := curr.GetFileHash(phash)
			if !fileutils.FileHashSame(oldFHash, newFHash) {
				logger.Error("restore-check-modifications",
					fmt.Sprintf("U %s\n      (%s =/=> %s)", relpath, oldFHash, newFHash),
					"\nLocal modifications found, please take a snapshot first.")
			}
		}
	}
}

func walk_root(hist *history.Hist) *history.Hist {
	rootpath := fileutils.CurrentWD()
	hist.SetMetaString("PWD", rootpath)

	filepath.WalkDir(rootpath, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			logger.Error("restore-walk-root", rootpath, "Failed to walk root directory.")
			return e
		}

		// ignore items here
		if d.Name() == fileutils.GetRootSettingsPath() {
			return nil
		}

		// add the files
		if !d.IsDir() {
			relpath, err := fileutils.CalcRelativePath(rootpath, s)
			if err != nil {
				logger.Error("restore-walk-root", s, "Failed to determine relative path.")
			}
			fhash, err := fileutils.CalcFileHash(s, d)
			if err != nil {
				logger.Error("restore-walk-root", s, "Failed to read file info.")
			}
			phash := fileutils.CalcPathHash(relpath)

			hist.AddPath(phash, relpath, d.Name(), fhash)
		}
		return nil
	})

	return hist
}

func calc_latest_ssid(remote string, rootname string) int {
	for i := 1000; i > 0; i-- {
		if fileutils.SSExists(i, remote, rootname) {
			return i
		}
	}
	return 0
}

// let LAST = last snapshot = 06 (say)

// For each path in the root,
// 	calc pathHash = md5(RelPath/Name)
// 	if PathHash in LAST,
// 		calc_file_hash()
// 		compare_with_last_hash()
// 		error if no match, must shot first, before restoring.
// 	if PathHash not in LAST,
// 		error, must shot first before restoring.
// 	if PathHash not in 02,
// 		Delete Path
// 		Add Action:
// 			Root>RelPath>D>PathHash>06>Name>Hash

// calculate_meta()
// sort_lines()
// print_actions()

// Foreach PathHash in 02,
// 	Create Actions;
// 	if CU,
// 		Copy PathHash/02 to WD/Path/Name
// 	if R,
// 		Copy PathHash/01 to WD/Path/Name
// 	if D,
// 		Delete WD/Path/Name

// Set last snapshot = 02 in .shot file
