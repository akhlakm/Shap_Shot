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
	rootname := settings.RootName()

	// load the last ss
	// can be 0 when new, or specific int
	lastss := settings.LastSnapshot()
	lastHistory := history.Make(lastss, remote, rootname)
	if lastss > 0 && !fileutils.SSExists(lastss, remote, rootname) {
		logger.Error("snapshot-load", fmt.Sprint(lastss),
			"No such snapshot exists, current settings might be invalid, "+
				"or remote files might be corrupted. "+
				"Please check remote files and rerun 'init'.")
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
	localHistory = calculate_meta_items(localHistory)

	logger.Print("\nChanges to commit:\n")
	localHistory.Print()

	if args.HasFlag("--dry") || args.HasFlag("-n") {
		// --dry has a higher priority over --go
		logger.Print("\nDry run. Snapshot is NOT restored.\n")
	} else if args.HasFlag("--go") || args.HasFlag("-go") {
		perform_actions(localHistory)
		settings.SetLastSnapshot(remoteHistory.SnapId)
		settings.Write()
	} else {
		logger.Print("\nDry run. Snapshot is NOT restored.\n")
	}
}

func perform_actions(loc *history.Hist) {
	count := 0
	rootpath := fileutils.CurrentWD()
	for phash := range loc.RelPath {
		crud := loc.GetCrud(phash)
		if crud == "C" || crud == "U" {
			relpath := loc.GetRelPath(phash)
			srcpath := loc.GetRestorePath(phash)
			dstpath := fileutils.PathJoin(rootpath, relpath)

			if !fileutils.FileExists(srcpath) {
				logger.Error("restore-action", srcpath, "File does not exists.")
			}
			logger.Trace("restore-copyfile", dstpath)
			cpbytes, err := fileutils.CopyFile(srcpath, dstpath)
			if err != nil {
				fmt.Println(err)
				logger.Error("restore-copyfile", srcpath, "Failed to copy file.")
			}
			count++
			logger.Print(fmt.Sprintf("OK -- %s (%d bytes)", relpath, cpbytes))
		}

		// We are not deleting anything.
	}
	logger.Print(fmt.Sprintf("DONE -- %d files copied", count))
}

func calculate_meta_items(hist *history.Hist) *history.Hist {
	create := hist.CountCrud("C")
	retain := hist.CountCrud("R")
	update := hist.CountCrud("U")
	delete := hist.CountCrud("D")
	total := create + retain + update
	hist.SetMetaInt("FileCount", total)

	// format crud: +9;=20;^2;-1
	crud := fmt.Sprintf("+%d;=%d;^%d;-%d", create, retain, update, delete)
	hist.SetMetaString("CRUD", crud)

	return hist
}

func calc_action_items(rem, loc *history.Hist) *history.Hist {
	// For each path in the root,
	for _, phash := range loc.PathHashList() {
		if !rem.IsPathHash(phash) {
			// 	if PathHash not in 02,
			// 		Delete Path
			loc.SetCrud(phash, "D")
		}
	}

	// Foreach PathHash in 02,
	for _, phash := range rem.PathHashList() {
		// similar local file exists
		if loc.IsPathHash(phash) {
			// 	if D, Delete WD/Path/Name
			if rem.GetCrud(phash) == "D" {
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
			// copy everything else that hasn't been deleted
			if rem.GetCrud(phash) != "D" {
				loc.SetAction(phash, rem.GetAction(phash))
				loc.SetCrud(phash, "C")
				remTarget := rem.GetTarget(phash)
				loc.SetTarget(phash, remTarget)
			}
		}
	}
	return loc
}

func check_local_modifications(last, new *history.Hist) {
	if last.SnapId == 0 {
		return
	}
	for _, phash := range new.PathHashList() {
		relpath := new.GetRelPath(phash)
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
			newFHash := new.GetFileHash(phash)
			if !fileutils.FileHashSame(oldFHash, newFHash) {
				logger.Error("restore-check-modifications", fmt.Sprintf("U %s (%s =/=> %s)", relpath, oldFHash, newFHash),
					"Local modifications found, please take a snapshot first.")
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
