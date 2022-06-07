package snapshot

import (
	"fmt"
	"io/fs"
	"os"
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
	lastss := settings.LastSnapshot()
	lastHistory := history.Make(lastss, remote, rootname)
	if lastss > 0 && !fileutils.SSExists(lastss, remote, rootname) {

		if !fileutils.DirExists(remote) {
			errmsg := "Remote directory does not exist.\n" +
				"\nMake sure it is mounted.\n"
			logger.Error("snapshot-execute", remote, errmsg)
		}

		logger.Error("snapshot-load", fmt.Sprint(lastss),
			"Last snapshot does not exist in remote.\n"+
				"\nRoot settings suggest existence of previous snapshots.\n"+
				"Please rerun 'init' if needed, or run 'list' to see the available snapshots in the remote.")
	}

	lastHistory.Load()
	// lastHistory.Print()

	// new history
	newss := calc_new_ssid(remote, rootname)
	newHistory := history.Make(newss, remote, rootname)
	newHistory = walk_root(newHistory)
	newHistory = compare(lastHistory, newHistory)
	newHistory = calculate_meta_items(newHistory)
	logger.Print("\nChanges to commit:\n")
	newHistory.Print()

	if args.HasFlag("--dry") || args.HasFlag("-n") {
		// --dry has a higher priority over --go
		logger.Print("\nDry run. Snapshot is NOT committed.")
		logger.Print("Please specify --go to commit the changes.")
	} else if args.HasFlag("--go") || args.HasFlag("-go") {
		perform_actions(newHistory)
		newHistory.Write()
		settings.SetLastSnapshot(newHistory.SnapId)
		settings.Write()
	} else {
		logger.Print("\nDry run. Snapshot is NOT committed.")
		logger.Print("Please specify --go to commit the changes.")
	}

}

func perform_actions(hist *history.Hist) {
	count := 0
	rootpath := fileutils.CurrentWD()
	for phash := range hist.RelPath {
		crud := hist.GetCrud(phash)
		if crud == "C" || crud == "U" {
			relpath := hist.GetRelPath(phash)
			srcpath := fileutils.PathJoin(rootpath, relpath)
			dstpath := hist.GetBackupPath(phash)

			if !fileutils.FileExists(srcpath) {
				logger.Error("snapshot-copyfile", srcpath, "File does not exists.")
			}
			logger.Trace("snapshot-copyfile", dstpath)
			// fmt.Println("copy :", srcpath, "==>", dstpath)
			cpbytes, err := fileutils.CopyFile(srcpath, dstpath)
			if err != nil {
				fmt.Println(err)
				logger.Error("snapshot-copyfile", srcpath, "Failed to copy file.")
			}
			count++
			logger.Print(fmt.Sprintf("OK -- %s (%d bytes)", relpath, cpbytes))
		}
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

	hist.SetMetaString("DATE", fileutils.GetTimeString())

	host, err := os.Hostname()
	if err == nil {
		hist.SetMetaString("HOST", host)
	}

	return hist
}

func compare(last, new *history.Hist) *history.Hist {
	for _, phash := range new.PathHashList() {
		if !last.IsPathHash(phash) {
			// 	C = If PathHash not in last
			new.SetCrud(phash, "C")
			new.SetTarget(phash, new.SnapId)
		} else {
			oldFHash := last.GetFileHash(phash)
			newFHash := new.GetFileHash(phash)
			if fileutils.FileHashSame(oldFHash, newFHash) {
				// 	R = If pathHash in 01 and FileHash same
				new.SetCrud(phash, "R")
				lastTarget := last.GetTarget(phash)
				new.SetTarget(phash, lastTarget)
			} else {
				// 	U = If PathHash in 01 and FileHash not same
				new.SetCrud(phash, "U")
				new.SetTarget(phash, new.SnapId)
			}
		}
	}

	// 	D = [for all PathHash:CRU in 01 not in 02]
	for _, phash := range last.PathHashList() {
		lastcrud := last.GetCrud(phash)
		// file was not deleted in the last ss
		if lastcrud != "D" {
			// no such file exist now
			if !new.IsPathHash(phash) {
				lf := last.GetAction(phash)
				new.SetAction(phash, lf)
				new.SetCrud(phash, "D")
				new.SetTarget(phash, last.SnapId)
			}
		}
	}

	return new
}

func walk_root(hist *history.Hist) *history.Hist {
	rootpath := fileutils.CurrentWD()
	hist.SetMetaString("ROOTDIR", rootpath)

	filepath.WalkDir(rootpath, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			logger.Error("snapshot-walk-root", rootpath, "Failed to walk root directory.")
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
				logger.Error("snapshot-walk-root", s, "Failed to determine relative path.")
			}
			fhash, err := fileutils.CalcFileHash(s, d)
			if err != nil {
				logger.Error("snapshot-walk-root", s, "Failed to read file info.")
			}
			phash := fileutils.CalcPathHash(relpath)

			hist.AddPath(phash, relpath, d.Name(), fhash)
			// fmt.Println(phash, relpath, d.Name(), fhash)
		}
		return nil
	})

	return hist
}

func calc_new_ssid(remote string, rootname string) int {
	for i := 1; i < 1000; i++ {
		if !fileutils.SSExists(i, remote, rootname) {
			return i
		}
	}
	logger.Error("snapshot-calc-new-ssid", remote+":"+rootname,
		"Cannot determine a new snapshot id.")
	return -1
}

// let LAST = last snapshot = 01 = read from .shot file.

// Date: 20/20/20 HH:MM:SS
// Desc: Blah blah blah
// PWD:  os.getcwd()
// HOST: $HOST
// CRUD: +9;=20;^2;-1
// Link: Root2>05
// Link: Root3>01

// # Count 6 '>' else throw error
// Root1>RelPath>Name>CU>PathHash>02>FileHash
// Root1>RelPath>Name>RD>PathHash>01>FileHash

// calculate_meta()
// sort_lines()
// print_lines()

// if not dry-run:
// 	Foreach PathHash in 02,
// 		if C|U:
// 			copyfile(Root/RelPath/Name, BackRoot/PathHash/02/Name)
// 	write_meta()
// 	write_lines()
