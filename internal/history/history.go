package history

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"snap/internal/fileutils"
	"snap/internal/logger"
	"strconv"
	"strings"
)

type FileItem struct {
	RelPath  string
	Name     string
	Target   int
	FileHash string
}

type Hist struct {
	Remote       string
	RootName     string
	SnapId       int
	SnapFilePath string
	Meta         map[string]string
	RelPath      map[string]string
	Name         map[string]string
	Target       map[string]int
	FileHash     map[string]string
	CRUD         map[string]string
}

func Make(ssid int, remote string, rootname string) *Hist {
	hist := &Hist{
		Remote:       remote,
		RootName:     rootname,
		SnapId:       ssid,
		SnapFilePath: fileutils.SSFilePath(ssid, remote, rootname),
		Meta:         make(map[string]string),
		RelPath:      make(map[string]string),
		Name:         make(map[string]string),
		Target:       make(map[string]int),
		FileHash:     make(map[string]string),
		CRUD:         make(map[string]string),
	}

	hist.SetMetaString("SSID", fileutils.FormatSnap(ssid))
	hist.SetMetaString("REMOTE", remote)
	hist.SetMetaString("ROOT", rootname)
	return hist
}

func (h *Hist) AddPath(pathHash string, relpath string, name string, filehash string) {
	h.RelPath[pathHash] = fileutils.PathNormalize(relpath)
	h.Name[pathHash] = name
	h.FileHash[pathHash] = filehash
}

func (h *Hist) PathHashList() []string {
	keys := make([]string, 0, len(h.Name))
	for key := range h.Name {
		keys = append(keys, key)
	}

	return keys
}

func (h *Hist) SnapFileExists() bool {
	return fileutils.SSExists(h.SnapId, h.Remote, h.RootName)
}

func (h *Hist) SnapFileOfNameExists(ssname string) bool {
	histDir := fileutils.SSHistoryDir(h.Remote, h.RootName)
	snapfile := fileutils.PathJoin(histDir, ssname)
	return fileutils.FileExists(snapfile)
}

func (h *Hist) IsPathHash(pathhash string) bool {
	if _, ok := h.Name[pathhash]; ok {
		return true
	}
	return false
}

func (h *Hist) GetCrud(pathHash string) string {
	val := h.CRUD[pathHash]
	return strings.ToUpper(val)
}

func (h *Hist) GetMeta(key string) string {
	if val, ok := h.Meta[key]; ok {
		return val
	}
	return "<none>"
}

func (h *Hist) GetName(pathHash string) string {
	val := h.Name[pathHash]
	return val
}

func (h *Hist) GetTarget(pathHash string) int {
	val := h.Target[pathHash]
	return val
}

// Relative path from the root
func (h *Hist) GetRelPath(pathHash string) string {
	val := h.RelPath[pathHash]
	return fileutils.PathNormalize(val)
}

// full path in the remote
func (h *Hist) GetBackupPath(phash string) string {
	backpath := fileutils.BackPath(h.Remote, h.RootName)
	fmtsnap := fileutils.FormatSnap(h.SnapId)
	filename := h.Name[phash]
	relbackpath := fileutils.PathJoin(backpath, phash, fmtsnap, filename)
	abspath, err := fileutils.AbsolutePath(relbackpath)
	if err != nil {
		logger.Error("history-backup-path", relbackpath, "Failed to calculate absolute path.")
	}
	return abspath
}

// full path in the remote
func (h *Hist) GetRestorePath(phash string) string {
	backpath := fileutils.BackPath(h.Remote, h.RootName)
	fmtsnap := fileutils.FormatSnap(h.GetTarget(phash))
	filename := h.Name[phash]
	relbackpath := fileutils.PathJoin(backpath, phash, fmtsnap, filename)
	abspath, err := fileutils.AbsolutePath(relbackpath)
	if err != nil {
		logger.Error("history-restore-path", relbackpath, "Failed to calculate absolute path.")
	}
	return abspath
}

func (h *Hist) GetFileHash(pathHash string) string {
	val := h.FileHash[pathHash]
	return val
}

func (h *Hist) GetAction(phash string) *FileItem {
	return &FileItem{
		Name:     h.Name[phash],
		RelPath:  h.RelPath[phash],
		Target:   h.Target[phash],
		FileHash: h.FileHash[phash],
	}
}

func (h *Hist) SetAction(phash string, fi *FileItem) {
	h.Name[phash] = fi.Name
	h.RelPath[phash] = fi.RelPath
	h.Target[phash] = fi.Target
	h.FileHash[phash] = fi.FileHash
}

func (h *Hist) CountCrud(crud string) int {
	total := 0
	for _, c := range h.CRUD {
		if c == crud {
			total += 1
		}
	}
	return total
}

func (h *Hist) SetCrud(pathhash string, crud string) {
	if len(crud) > 1 {
		logger.Error("history-set-crud", crud, "CRUD must be single character")
	}
	h.CRUD[pathhash] = strings.ToUpper(crud)
}

func (h *Hist) SetFileHash(pathhash string, hash string) {
	h.FileHash[pathhash] = hash
}

func (h *Hist) SetTarget(pathhash string, target int) {
	h.Target[pathhash] = target
}

func (h *Hist) SetMetaString(key string, value string) {
	h.Meta[key] = value
}

func (h *Hist) SetMetaInt(key string, value int) {
	h.Meta[key] = strconv.Itoa(value)
}

func (h *Hist) AddMetaInt(key string, value int) {
	if _, ok := h.Meta[key]; !ok {
		h.Meta[key] = "0"
	}
	prev, err := strconv.Atoi(h.Meta[key])
	if err != nil {
		logger.Error("history-add-meta", key, "Cannot parse meta as an integer to add to.")
	}
	h.Meta[key] = strconv.Itoa(prev + value)
}

func (h *Hist) get_action_string(phash string) string {
	// Root1>RelPath>CU>PathHash>02>Name>FileHash
	return fmt.Sprintf("    %s > %s > %s > %s > %04d > %s > %s",
		h.RootName,
		h.RelPath[phash],
		strings.ToUpper(h.CRUD[phash]),
		phash,
		h.Target[phash],
		h.Name[phash],
		h.FileHash[phash])
}

func (h *Hist) formatted_action_string(phash string) string {
	// Root1>RelPath>CU>PathHash>02>Name>FileHash
	return fmt.Sprintf("  %s > %s\n      FileHash: %s\n      LastSnapshot: %s > %04d > %s\n",
		h.RootName,
		h.RelPath[phash],
		h.FileHash[phash],
		phash,
		h.Target[phash],
		h.Name[phash])
}

func (h *Hist) Print() {
	h.PrintCrud("R")
	h.PrintCrud("C")
	h.PrintCrud("U")
	h.PrintCrud("D")
	h.PrintMeta()
}

func (h *Hist) PrintCrud(crud string) {
	logger.Print(crud + " ----------------------------------------------------")
	for phash := range h.RelPath {
		if h.GetCrud(phash) == strings.ToUpper(crud) {
			logger.Print(h.formatted_action_string(phash))
		}
	}
}

func (h *Hist) PrintMeta() {
	logger.Print("M ----------------------------------------------------")
	for key, val := range h.Meta {
		logger.Print(fmt.Sprintf("    %s\t=\t%s", key, val))
	}
	logger.Print("------------------------------------------------------")
}

func (h *Hist) Write() {
	lines := []string{}

	for key, val := range h.Meta {
		lines = append(lines, fmt.Sprintf("%s\t=\t%s", key, val))
	}

	for phash := range h.RelPath {
		line := h.get_action_string(phash)
		line = strings.TrimSpace(line)
		lines = append(lines, line)
	}

	snapfile := fileutils.SSFilePath(h.SnapId, h.Remote, h.RootName)
	err := fileutils.CreateParent(snapfile)
	if err != nil {
		log.Fatalf("failed creating directory: %s", err)
	}

	file, err := os.OpenFile(snapfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	datawriter := bufio.NewWriter(file)
	for _, data := range lines {
		_, _ = datawriter.WriteString(data + "\n")
	}

	datawriter.Flush()
	file.Close()
}

// Load and parse history file
func (h *Hist) Load() {
	if h.SnapId == 0 {
		return
	}
	snapfile := fileutils.SSFilePath(h.SnapId, h.Remote, h.RootName)
	logger.Trace("history-load", snapfile)

	file, err := os.Open(snapfile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		n := len(line)
		if n == 0 {
			continue
		}
		if strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			key := strings.TrimSpace(parts[0])
			val := strings.Join(parts[1:], "=")
			val = strings.TrimSpace(val)
			h.SetMetaString(key, val)
		} else if strings.Contains(line, ">") {
			// Root1>RelPath>CU>PathHash>02>Name>FileHash
			// Count 6 '>' else throw error
			parts := strings.Split(line, ">")
			if len(parts) < 7 {
				errmsg := "Snapshot file unreadable.\n" +
					"\nPlease make sure you have not manually edited the shot files in the history/ directory.\n" +
					"Expected format: Root1>RelPath>CU>PathHash>02>Name>FileHash\n"
				logger.Error("history-load", line, errmsg)
			}
			if h.RootName != strings.TrimSpace(parts[0]) {
				errmsg := "Snapshot was taken under a different root or different rootname.\n" +
					"\nPlease make sure you initialized with the same rootname.\n" +
					"If you want, you can manually rename the root directory in the remote.\n"
				logger.Error("history-load", parts[0], errmsg)
			}

			relpath := strings.TrimSpace(parts[1])
			crud := strings.TrimSpace(parts[2])
			pathhash := strings.TrimSpace(parts[3])
			target := strings.TrimSpace(parts[4])
			name := strings.TrimSpace(parts[5])
			filehash := strings.TrimSpace(parts[6])

			h.AddPath(pathhash, relpath, name, filehash)
			h.SetCrud(pathhash, crud)
			itarget, err := strconv.ParseInt(target, 10, 0)
			if err != nil {
				fmt.Println(line, "\n", err)
				logger.Error("history-load", target, "Not a valid snapshot target, shot file unreadable.")
			}
			h.SetTarget(pathhash, int(itarget))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	logger.Done("history-load", snapfile)
}

// Load and parse the initial meta section of the history file
func (h *Hist) LoadFileMeta(ssname string) {
	histDir := fileutils.SSHistoryDir(h.Remote, h.RootName)
	snapfile := fileutils.PathJoin(histDir, ssname)
	logger.Trace("history-load-meta", snapfile)

	file, err := os.Open(snapfile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		n := len(line)
		if n == 0 {
			continue
		}
		if strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			key := strings.TrimSpace(parts[0])
			val := strings.Join(parts[1:], "=")
			val = strings.TrimSpace(val)
			h.SetMetaString(key, val)

			if key == "SSID" {
				h.SnapId, err = strconv.Atoi(val)
				if err != nil {
					logger.Error("history-load-meta", val, "Invalid SSID in the snapshot file!")
				}
			}
		} else if strings.Contains(line, ">") {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	logger.Done("history-load-meta", snapfile)
}
