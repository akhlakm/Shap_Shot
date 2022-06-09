package fileutils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const root_settings_name string = ".shot-settings"
const back_snap_format string = "_%04d"
const back_files_directory string = "files"
const back_hist_directory string = "history"
const back_snap_file_format string = "%04d.shot"

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func DirExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func SSExists(ssid int, remote string, rootname string) bool {
	ssname := FormatSnapFile(ssid)
	fullpath := PathJoin(remote, rootname, back_hist_directory, ssname)
	return FileExists(fullpath)
}

func SSFilePath(ssid int, remote string, rootname string) string {
	ssname := FormatSnapFile(ssid)
	return PathJoin(remote, rootname, back_hist_directory, ssname)
}

func SSHistoryDir(remote, rootname string) string {
	return PathJoin(remote, rootname, back_hist_directory)
}

func BackPath(remote string, rootname string) string {
	return PathJoin(remote, rootname, back_files_directory)
}

// Production, executable path
// func CurrentWD() string {
// 	exepath, err := os.Executable()
// 	if err != nil {
// 		cwd, err2 := os.Getwd()
// 		if err2 != nil {
// 			log.Fatalln("Cannot determine current working directory!", err, err2)
// 		}
// 		return PathNormalize(cwd)
// 	}
// 	return PathNormalize(filepath.Dir(exepath))
// }

// Debug
func CurrentWD() string {
	cwd, err2 := os.Getwd()
	if err2 != nil {
		log.Fatalln("Cannot determine current working directory!", err2)
	}
	return PathNormalize(cwd)
}

// temporary checkout directory
func ShotPath(path string) string {
	return PathJoin(CurrentWD(), "_.shot", path)
}

func CalcPathMd5(path string) string {
	hash := md5.New()
	for i := 0; i < len(path); i++ {
		char := path[i]
		// get rid of path characters
		if char == '/' || char == '\\' || char == ':' {
			char = '>'
		}
		hash.Write([]byte{char})
	}

	hashInBytes := hash.Sum(nil)
	hashStr := hex.EncodeToString(hashInBytes)

	return hashStr
}

func CalcPathHash(relpath string) string {
	relpath = PathNormalize(relpath)
	// get the dir
	// dir, _ := filepath.Split(relpath)
	// return CalcPathMd5(relpath)
	return relpath
}

func CalcFileHash(fullpath string, d fs.DirEntry) (string, error) {
	hash := ""
	finfo, err := d.Info()
	if err != nil {
		return "", err
	}
	size := strconv.FormatInt(finfo.Size(), 10)
	modt := finfo.ModTime().Format("2006-01-02 03:04:05PM UTC-07:00")
	hash = size + "; " + modt
	return hash, nil
}

func FileHashSame(hash1, hash2 string) bool {
	return hash1 == hash2
}

func FileSizeSame(filehash string, size int64) bool {
	// hash = size + "; " + modt
	sizeInHash := strings.Split(filehash, ";")[0]
	return strings.TrimSpace(sizeInHash) == fmt.Sprint(size)
}

func CalcRelativePath(basepath string, fullpath string) (string, error) {
	return filepath.Rel(basepath, fullpath)
}

func IsASubPath(parent, child string) bool {
	// get abspath first
	parent, _ = AbsolutePath(parent)
	child, _ = AbsolutePath(child)

	// normalize
	parent = PathNormalize(parent)
	child = PathNormalize(child)

	// calc parent path to child path relation
	relation, err := filepath.Rel(parent, child)

	// cannot determine such relation
	if err != nil {
		return false
	} else {
		if strings.Contains(relation, "..") {
			return false
		} else {
			return true
		}
	}
}

func FormatSnap(id int) string {
	return fmt.Sprintf(back_snap_format, id)
}

func FormatSnapFile(id int) string {
	return fmt.Sprintf(back_snap_file_format, id)
}

func GetTimeString() string {
	//formatting works in a one-two-three... pattern
	//in the date portion, 01 is the month (American format) and 02 is the day and 06 is the year
	//in the time portion, 03 or 15 is the hour and 04 is the minutes while 05 is the seconds
	//at the end the UTC offset will always begin with - (negative) and 0700 [-0700]
	// Example: "Mon 01/02/06 03:04:05PM -07:00"
	return time.Now().Local().Format("Mon 2006-01-02 03:04:05PM -07:00 UTC")
}

func GetRootSettingsPath() string {
	// @todo: back navigate to top to find any existing
	// settings file in a parent directory.
	return PathNormalize(root_settings_name)
}

func PathJoin(elem ...string) string {
	return filepath.Join(elem...)
}

// Convert path to forward slash
// go works well with forward slash everywhere
func PathNormalize(path string) string {
	path = filepath.FromSlash(path)
	res := ""
	for i := 0; i < len(path); i++ {
		char := path[i]
		if char == '\\' {
			char = '/'
		}
		res += string(char)
	}
	return res
}

func AbsolutePath(path string) (string, error) {
	return filepath.Abs(path)
}

// Copy to a temp file first, then rename.
// Create the parent dir, if not exist.
func CopyFile(src, dst string) (int64, error) {
	if src == dst {
		return 0, nil
	}
	in, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("couldn't open source file: %s", err)
	}

	// create the parent directory
	err = CreateDirectory(filepath.Dir(dst))
	if err != nil {
		in.Close()
		return 0, err
	}

	tmpfile := dst + ".tmp"
	tmp, err := os.OpenFile(tmpfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		in.Close()
		return 0, fmt.Errorf("couldn't open dest tmpfile: %s", err)
	}

	bytesWritten, err := io.Copy(tmp, in)
	in.Close()
	if err != nil {
		return bytesWritten, fmt.Errorf("writing to dest tmpfile failed: %s", err)
	}

	// flush
	err = tmp.Sync()
	if err != nil {
		return bytesWritten, fmt.Errorf("tmpfile flush error: %s", err)
	}
	tmp.Close()

	sfinfo, err := os.Stat(src)
	if err != nil {
		return bytesWritten, fmt.Errorf("coundn't stat srcfile: %s", err)
	}

	// check if copy was okay
	if bytesWritten != sfinfo.Size() {
		return bytesWritten, fmt.Errorf("coundn't copy all bytes, src %d bytes, copied %d bytes",
			sfinfo.Size(), bytesWritten)
	}

	// copy the modification time
	err = os.Chtimes(tmpfile, sfinfo.ModTime(), sfinfo.ModTime())
	if err != nil {
		return bytesWritten, fmt.Errorf("coundn't chtimes tmpfile: %s", err)
	}

	// rename the temp file
	err = os.Rename(tmpfile, dst)
	if err != nil {
		return bytesWritten, fmt.Errorf("coundn't rename tmpfile: %s", err)
	}

	return bytesWritten, nil
}

func DeleteFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("failed to remove file: %s", err)
	}
	return nil
}

func CreateParent(fpath string) error {
	err := CreateDirectory(filepath.Dir(fpath))
	if err != nil {
		return err
	}
	return nil
}

func CreateDirectory(path string) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory: %s", err)
	}
	return nil
}

// get_files_path(remote, root)
// get_hist_path(remote, root)
// calc_path_hash()
// calc_file_hash()
// copy_file()
// delete_file()
// create_directory()
// pathjoin()
// UnixPath()
