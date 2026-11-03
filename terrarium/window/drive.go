package window

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// The driver's side: `lizardscript drive [--pid N] Command args...` finds
// the drive file of a running window and sends it one command.

// driveInfo is what a drive file holds.
type driveInfo struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
	PID   int    `json:"pid"`
	Title string `json:"title"`
}

// configDir is the folder of the user's settings; tests point it at a
// folder of their own.
var configDir = os.UserConfigDir

// driveDir is the folder of the drive files: <config dir>/lizard, next to
// the pen file.
func driveDir() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lizard"), nil
}

// DriveFile is the drive file of the program with process id pid in dir.
// Each program has its own, so a second one never takes over the first's.
func DriveFile(dir string, pid int) string {
	return filepath.Join(dir, fmt.Sprintf("drive-%d.json", pid))
}

// driveFiles are the drive files in dir, with their process ids.
func driveFiles(dir string) map[string]int {
	out := map[string]int{}
	names, _ := filepath.Glob(filepath.Join(dir, "drive-*.json"))
	for _, name := range names {
		id := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(name), "drive-"), ".json")
		if pid, err := strconv.Atoi(id); err == nil {
			out[name] = pid
		}
	}
	return out
}

// cleanStale removes the drive files of programs that no longer run.
func cleanStale(dir string) {
	for name, pid := range driveFiles(dir) {
		if !pidAlive(pid) {
			os.Remove(name)
		}
	}
}

// findDrive reads the drive file of the program pid, or of the newest one
// running when pid is 0.
func findDrive(pid int) (driveInfo, error) {
	dir, err := driveDir()
	if err != nil {
		return driveInfo{}, err
	}
	best, bestTime := "", int64(0)
	for name, p := range driveFiles(dir) {
		if (pid != 0 && p != pid) || !pidAlive(p) {
			continue
		}
		st, err := os.Stat(name)
		if err != nil {
			continue
		}
		if t := st.ModTime().UnixNano(); best == "" || t > bestTime {
			best, bestTime = name, t
		}
	}
	if best == "" {
		if pid != 0 {
			return driveInfo{}, fmt.Errorf("no automated window runs with pid %d", pid)
		}
		return driveInfo{}, fmt.Errorf("no automated window runs (a program turns it on with window.automation, or %s=1)", AutomationEnv)
	}
	data, err := os.ReadFile(best)
	if err != nil {
		return driveInfo{}, err
	}
	var info driveInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return driveInfo{}, fmt.Errorf("can't read %s: %v", best, err)
	}
	return info, nil
}

// DriveUsage is how `lizardscript drive` is used.
const DriveUsage = "usage: lizardscript drive [--pid N] Command [args...] (try List)"

// Drive sends one command to the newest automated window (or the one of
// --pid N) and writes its JSON reply to out. An error reply is written too,
// and is Drive's error.
func Drive(args []string, out io.Writer) error {
	pid := 0
	if len(args) >= 2 && args[0] == "--pid" {
		n, err := strconv.Atoi(args[1])
		if err != nil || n <= 0 {
			return fmt.Errorf("--pid wants a process id, got %q", args[1])
		}
		pid, args = n, args[2:]
	}
	if len(args) < 1 {
		return fmt.Errorf(DriveUsage)
	}
	info, err := findDrive(pid)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(request{Cmd: args[0], Args: args[1:]})
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/cmd", info.Port), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set(TokenHeader, info.Token)
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	reply, _ := io.ReadAll(resp.Body)
	out.Write(reply)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s failed", args[0])
	}
	return nil
}
