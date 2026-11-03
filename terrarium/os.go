package terrarium

import (
	"lizardscript/bundle"
	"lizardscript/object"
	"lizardscript/terrarium/internal/arg"
	"os"
	"path/filepath"
	"runtime"
)

// os: the program's arguments and environment, and the system it runs on.

// Exit ends the program for os.exit. A host can replace it, to close its
// own things first.
var Exit = os.Exit

// ProgramFile is the path of the program's main file, for
// os.program_dir(): a path on disk, a path in a built program
// ("bundle:/main.liz"), or "" when there is no file, like in the REPL. A
// host sets it before the program runs.
var ProgramFile string

func osModule() *object.Hash {
	return arg.Module("os", map[string]object.BuiltinFunction{
		"args":        osArgs,
		"env":         osEnv,
		"exit":        osExit,
		"platform":    osPlatform,
		"locale":      osLocale,
		"config_dir":  osConfigDir,
		"program_dir": osProgramDir,
		"reveal":      osReveal,
		"open_url":    osOpenURL,
		"memory":      osMemory,
	})
}

// os.args() is the program's arguments as strings, without its own name.
func osArgs(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	out := &object.Array{Elements: make([]object.Object, len(Args))}
	for i, a := range Args {
		out.Elements[i] = arg.Str(a)
	}
	return out
}

// os.env(name) is the value of the environment variable, or null when it
// isn't set.
func osEnv(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 1); err != nil {
		return err
	}
	name, err := arg.String(args, 0, "name")
	if err != nil {
		return err
	}
	value, ok := os.LookupEnv(name)
	if !ok {
		return arg.Null
	}
	return arg.Str(value)
}

// os.exit(code = 0) ends the program now, with the exit code.
func osExit(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 1); err != nil {
		return err
	}
	code := 0
	if len(args) == 1 {
		n, err := arg.IntIn(args, 0, "code", 0, 255)
		if err != nil {
			return err
		}
		code = n
	}
	Exit(code)
	return arg.Null
}

// Platform is the name of the system: "macos", "windows" or "linux" (or
// Go's own name for another one).
func Platform() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}

// os.platform() is Platform.
func osPlatform(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	return arg.Str(Platform())
}

// os.locale() is the user's language as a lowercase two-letter code.
func osLocale(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	return arg.Str(Locale())
}

// os.config_dir() is the folder where programs keep the user's settings
// (~/Library/Application Support on macOS, %AppData% on Windows,
// ~/.config on Linux). It doesn't make it.
func osConfigDir(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return arg.Errorf("can't find the settings folder: %v", err)
	}
	return arg.Str(filepath.ToSlash(dir))
}

// Locale is the user's language as a lowercase two-letter code, like "es".
// It asks the system (the macOS preferences, the Windows locale), then
// LC_ALL and LANG, and is "en" when none of them says.
func Locale() string {
	if code := platformLocale(); code != "" {
		return code
	}
	for _, name := range []string{"LC_ALL", "LANG"} {
		if code := twoLetters(os.Getenv(name)); code != "" {
			return code
		}
	}
	return "en"
}

// twoLetters is the language code at the start of s, like "es" from
// "es_CO.UTF-8", or "" when s doesn't start with one.
func twoLetters(s string) string {
	if len(s) < 2 {
		return ""
	}
	a, b := s[0]|0x20, s[1]|0x20
	if a < 'a' || a > 'z' || b < 'a' || b > 'z' {
		return ""
	}
	if len(s) > 2 && s[2]|0x20 >= 'a' && s[2]|0x20 <= 'z' { // "eng" is not a two-letter code
		return ""
	}
	return string([]byte{a, b})
}

// os.program_dir() is the folder of the program's main file, so a program
// finds the files that come with it: an absolute path, or "bundle:" in a
// program built with lizardscript build, whose files fs reads there. With
// no file (the REPL) it is the current folder.
func osProgramDir(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	if bundle.IsPath(ProgramFile) {
		return arg.Str(bundle.Root)
	}
	dir, err := filepath.Abs(filepath.Dir(ProgramFile))
	if err != nil {
		return arg.Errorf("can't find the program's folder: %v", err)
	}
	return arg.Str(filepath.ToSlash(dir))
}

// os.reveal(path) shows the file or folder in the file manager: selected
// in Finder or Explorer, its folder on Linux.
func osReveal(args ...object.Object) object.Object {
	s, err := pathArgs(args, "path")
	if err != nil {
		return err
	}
	if rerr := Reveal(s[0]); rerr != nil {
		return systemError("show", s[0], rerr)
	}
	return arg.Null
}

// os.open_url(url) opens an http, https or mailto address in the user's
// browser or mail program.
func osOpenURL(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 1); err != nil {
		return err
	}
	url, err := arg.String(args, 0, "url")
	if err != nil {
		return err
	}
	if oerr := OpenURL(url); oerr != nil {
		return arg.Errorf("can't open %s: %v", url, oerr)
	}
	return arg.Null
}

// os.memory() is the machine's memory in bytes: {"total": all of it,
// "free": what can be used now}.
func osMemory(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	total, free, merr := Memory()
	if merr != nil {
		return arg.Errorf("can't read the memory: %v", merr)
	}
	h := object.NewHash()
	arg.Set(h, "total", arg.Int64(int64(total)))
	arg.Set(h, "free", arg.Int64(int64(free)))
	return h
}
