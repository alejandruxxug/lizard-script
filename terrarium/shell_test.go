package terrarium

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fakeShell replaces the programs the shell runs, and records them.
func fakeShell(t *testing.T, fail bool) *[]string {
	t.Helper()
	var calls []string
	saved := runCommand
	runCommand = func(name string, args ...string) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if fail {
			return errors.New("no")
		}
		return nil
	}
	t.Cleanup(func() { runCommand = saved })
	return &calls
}

// fakeTrash makes the Trash a folder of the test.
func fakeTrash(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "Trash", "files")
	saved := trashDir
	trashDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { trashDir = saved })
	return dir
}

func TestTrash(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Recycle Bin has no folder to look in")
	}
	trash := fakeTrash(t)
	fakeShell(t, true)
	dir := filepath.ToSlash(t.TempDir())
	d := quote(dir)
	check(t, []testCase{
		{"a file", `let p = fs.join(` + d + `, "a.txt"); fs.write(p, "x"); fs.trash(p); fs.exists(p)`, "false", false},
		{"twice the same name", `let p = fs.join(` + d + `, "b.txt"); fs.write(p, "1"); fs.trash(p); fs.write(p, "2"); fs.trash(p); fs.exists(p)`, "false", false},
		{"a folder", `let p = fs.join(` + d + `, "f"); fs.mkdir(fs.join(p, "in")); fs.trash(p); fs.exists(p)`, "false", false},
		{"missing", `fs.trash(fs.join(` + d + `, "nope"))`, "can't move " + dir + "/nope to the Trash: no such file or directory", true},
	})
	for _, name := range []string{"a.txt", "b.txt", "b 2.txt", "f"} {
		if _, err := os.Stat(filepath.Join(trash, name)); err != nil {
			t.Errorf("%s is not in the Trash: %v", name, err)
		}
	}
}

func TestTrashAsksTheSystem(t *testing.T) {
	saved := trashDir
	trashDir = func() (string, error) { return "", errNoTrashDir }
	defer func() { trashDir = saved }()
	calls := fakeShell(t, false)
	path := filepath.Join(t.TempDir(), "x")
	os.WriteFile(path, nil, 0o644)
	if err := Trash(path); err != nil || len(*calls) == 0 && runtime.GOOS != "windows" {
		// Windows asks PowerShell through runCommandEnv, not runCommand.
		if runtime.GOOS != "windows" {
			t.Fatalf("Trash: %v, calls %v", err, *calls)
		}
	}
	failing := fakeShell(t, true)
	if err := Trash(path); err == nil || runtime.GOOS != "windows" && len(*failing) == 0 {
		t.Fatalf("a Trash that fails gave %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("the file is gone after the Trash failed")
	}
}

func TestTrashInfo(t *testing.T) {
	when := time.Date(2026, 11, 3, 9, 5, 7, 0, time.Local)
	got := trashInfoText("/home/liz/My File#1.txt", when)
	want := "[Trash Info]\nPath=/home/liz/My%20File%231.txt\nDeletionDate=2026-11-03T09:05:07\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRevealAndOpenURL(t *testing.T) {
	calls := fakeShell(t, false)
	dir := filepath.ToSlash(t.TempDir())
	check(t, []testCase{
		{"reveal", `os.reveal(` + quote(dir) + `)`, "null", false},
		{"open a url", `os.open_url("https://example.com/?a=1&b=2")`, "null", false},
		{"mailto", `os.open_url("mailto:liz@example.com")`, "null", false},
		{"reveal missing", `os.reveal(` + quote(dir+"/nope") + `)`, "can't show " + dir + "/nope", true},
		{"only web addresses", `os.open_url("/bin/sh")`, "can't open /bin/sh: only http, https and mailto addresses can be opened", true},
		{"a file url", `os.open_url("file:///etc/passwd")`, "only http, https and mailto", true},
	})
	joined := strings.Join(*calls, "\n")
	if !strings.Contains(joined, "https://example.com/?a=1&b=2") || !strings.Contains(joined, "mailto:liz@example.com") {
		t.Errorf("the shell was asked: %s", joined)
	}
}

func TestMemory(t *testing.T) {
	check(t, []testCase{
		{"keys", `keys(os.memory())`, `["total", "free"]`, false},
		{"sizes", `let m = os.memory(); m.total > 0 && m.free >= 0 && m.free <= m.total`, "true", false},
		{"takes nothing", `os.memory(1)`, "want 0, got 1", true},
	})
}
