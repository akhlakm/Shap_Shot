package settings

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

type Settings struct {
	root    map[string]string
	remotes map[string]string
	file    string
}

var initialized *Settings = nil

func Create(rootname string, remotepath string) {
	if initialized == nil {
		initialized = &Settings{
			root:    make(map[string]string),
			remotes: make(map[string]string),
			file:    fileutils.GetRootSettingsPath(),
		}
	}

	initialized.root["name"] = rootname
	initialized.root["snapshot"] = "0"
	initialized.remotes["default"] = fileutils.PathNormalize(remotepath)
}

func DefaultRemote() string {
	if _, ok := initialized.remotes["default"]; !ok {
		logger.Error("settings-default-remote", "",
			"No 'default' remote exists in the settings file.\n"+
				"\nPlease make sure the remotes section contains a default field to a remote path."+
				"\nOr, run init again.")
	}

	return initialized.remotes["default"]
}

func RootName() string {
	if _, ok := initialized.root["name"]; !ok {
		logger.Error("settings-root-name", "",
			"No root name in the settings file.\n"+
				"\nPlease make sure the root section contains a name field for the current directory."+
				"\nOr, run init again.")
	}
	return initialized.root["name"]
}

func LastSnapshot() int {
	if _, ok := initialized.root["snapshot"]; !ok {
		logger.Error("settings-last-snapshot", "",
			"No snapshot number in the settings file.\n"+
				"\nPlease make sure the settings file contains a valid snapshot number."+
				"\nOr, run init again.")
	}

	ss, err := strconv.Atoi(initialized.root["snapshot"])
	if err != nil {
		logger.Error("settings-last-snapshot", "",
			"Invalid snapshot number in the settings file.\n"+
				"\nPlease make sure the settings file contains a valid snapshot number."+
				"\nOr, run init again.")
	}

	return ss
}

func SetLastSnapshot(ssid int) {
	initialized.root["snapshot"] = strconv.Itoa(ssid)
}

func Write() {
	if initialized != nil {
		initialized.write()
	} else {
		logger.Error("settings-write", "", "Settings not initialized")
	}
}

func Exists() bool {
	file := fileutils.GetRootSettingsPath()
	return fileutils.FileExists(file)
}

func Load() {
	if initialized == nil {
		initialized = &Settings{
			root:    make(map[string]string),
			remotes: make(map[string]string),
			file:    fileutils.GetRootSettingsPath(),
		}
	}
	if fileutils.FileExists(initialized.file) {
		initialized.read()
	} else {
		logger.Error("settings-load", initialized.file,
			"No settings file found.\n"+
				"\nPlease run 'init' with a rootname (a name for the current project directory),\n"+
				"and a path to a remote folder to backup to.\n"+
				"\nUSAGE: init <rootname> <remote folder path>\n")
	}
}

func (s *Settings) write() {
	logger.Trace("settings-write", s.file)
	file, err := os.OpenFile(s.file, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer file.Close()

	datawriter := bufio.NewWriter(file)

	// Write root section title
	datawriter.WriteString("[ROOT]\n")
	// Write root key values
	for k, v := range s.root {
		datawriter.WriteString(fmt.Sprintf("%s = %s\n", k, v))
	}

	// Write remotes section title
	datawriter.WriteString("\n[REMOTES]\n")
	// Write remote name = path
	for k, v := range s.remotes {
		datawriter.WriteString(fmt.Sprintf("%s = %s\n", k, v))
	}

	datawriter.Flush()
	logger.Done("settings-write", s.file)
}

func (s *Settings) read() {
	logger.Trace("read-settings", s.file)
	if fileutils.FileExists(s.file) {
		file, err := os.Open(s.file)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		section := "MAIN"

		for scanner.Scan() {
			line := scanner.Text()
			line = strings.TrimSpace(line)
			n := len(line)
			if n == 0 {
				continue
			}
			if line[0] == '[' && line[n-1] == ']' {
				section = line[1 : n-1]
				section = strings.ToUpper(section)
			} else if strings.Contains(line, "=") {
				parts := strings.Split(line, "=")
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				k = strings.ToLower(k)
				if section == "ROOT" {
					s.root[k] = v
				} else if section == "REMOTES" {
					s.remotes[k] = fileutils.PathNormalize(v)
				}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		// Sanity check
		// if _, ok := s.root["name"]; !ok {
		// 	fmt.Print(s.root)
		// 	logger.Error("read-settings", s.file, "No root 'name' exists in the settings file.")
		// }
		// if _, ok := s.root["snapshot"]; !ok {
		// 	logger.Error("read-settings", s.file, "No root 'snapshot' exists in the settings file.")
		// }
		// if _, ok := s.remotes["default"]; !ok {
		// 	logger.Error("read-settings", s.file, "No 'default' remote exists in the settings file.")
		// }

		logger.Done("read-settings", "")
	} else {
		logger.Error("read-settings", s.file, "Directory does not contain a settings file.")
	}

}

// read_settings():
// 	root[key] =
// 	remotes[key] =
// 	if [ ] in line,
// 		section = root or remotes
// 	if = in line,
// 		section[key] = value
// 		parse_remote_as_unix_path()
