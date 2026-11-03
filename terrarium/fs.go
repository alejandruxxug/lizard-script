package terrarium

import (
	"errors"
	"io/fs"
	"lizardscript/object"
	"lizardscript/terrarium/internal/arg"
	"os"
	"path/filepath"
	"sort"
)

// fs: files and folders. Paths come back with "/" between their parts on
// every system (Windows takes it too), so a program never has to know which
// one it runs on.

func fsModule() *object.Hash {
	return arg.Module("fs", map[string]object.BuiltinFunction{
		"read":   fsRead,
		"write":  fsWrite,
		"append": fsAppend,
		"exists": fsExists,
		"list":   fsList,
		"mkdir":  fsMkdir,
		"remove": fsRemove,
		"home":   fsHome,
		"join":   fsJoin,
	})
}

// systemError is the message of an error from the system, without Go's
// own wording: can't read notes.txt: no such file or directory.
func systemError(verb, path string, err error) *object.Error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		err = linkErr.Err
	}
	return arg.Errorf("can't %s %s: %v", verb, path, err)
}

// pathArgs checks that a builtin got count arguments, all strings: the path
// first, named in messages by what.
func pathArgs(args []object.Object, what ...string) ([]string, *object.Error) {
	if err := arg.Count(args, len(what), len(what)); err != nil {
		return nil, err
	}
	out := make([]string, len(what))
	for i := range what {
		s, err := arg.String(args, i, what[i])
		if err != nil {
			return nil, err
		}
		out[i] = s
	}
	return out, nil
}

// fs.read(path) is the text of the file.
func fsRead(args ...object.Object) object.Object {
	s, err := pathArgs(args, "path")
	if err != nil {
		return err
	}
	data, rerr := os.ReadFile(s[0])
	if rerr != nil {
		return systemError("read", s[0], rerr)
	}
	return arg.Str(string(data))
}

// fs.write(path, text) replaces the file with text, or makes it. Another
// program never sees half of it: the text goes to a new file next to it,
// which then takes its place.
func fsWrite(args ...object.Object) object.Object {
	s, err := pathArgs(args, "path", "text")
	if err != nil {
		return err
	}
	if werr := WriteFileAtomic(s[0], []byte(s[1])); werr != nil {
		return systemError("write", s[0], werr)
	}
	return arg.Null
}

// fs.append(path, text) adds text at the end of the file, or makes it.
func fsAppend(args ...object.Object) object.Object {
	s, err := pathArgs(args, "path", "text")
	if err != nil {
		return err
	}
	file, oerr := os.OpenFile(s[0], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if oerr != nil {
		return systemError("append to", s[0], oerr)
	}
	_, werr := file.WriteString(s[1])
	if cerr := file.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return systemError("append to", s[0], werr)
	}
	return arg.Null
}

// fs.exists(path) says whether there is a file or folder at path.
func fsExists(args ...object.Object) object.Object {
	s, err := pathArgs(args, "path")
	if err != nil {
		return err
	}
	_, serr := os.Stat(s[0])
	return arg.Bool(serr == nil)
}

// fs.list(dir) is the names of what is in the folder, files and folders,
// in alphabetical order.
func fsList(args ...object.Object) object.Object {
	s, err := pathArgs(args, "folder")
	if err != nil {
		return err
	}
	entries, rerr := os.ReadDir(s[0])
	if rerr != nil {
		return systemError("list", s[0], rerr)
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	sort.Strings(names)
	out := &object.Array{Elements: make([]object.Object, len(names))}
	for i, name := range names {
		out.Elements[i] = arg.Str(name)
	}
	return out
}

// fs.mkdir(path) makes the folder and the ones above it that are missing.
// A folder that is already there is fine.
func fsMkdir(args ...object.Object) object.Object {
	s, err := pathArgs(args, "path")
	if err != nil {
		return err
	}
	if merr := os.MkdirAll(s[0], 0o755); merr != nil {
		return systemError("make", s[0], merr)
	}
	return arg.Null
}

// fs.remove(path) deletes a file or an empty folder. It refuses a folder
// with something in it, so a mistake can't delete a whole tree.
func fsRemove(args ...object.Object) object.Object {
	s, err := pathArgs(args, "path")
	if err != nil {
		return err
	}
	if info, serr := os.Stat(s[0]); serr == nil && info.IsDir() {
		if entries, rerr := os.ReadDir(s[0]); rerr == nil && len(entries) > 0 {
			return arg.Errorf("can't remove %s: the folder is not empty", s[0])
		}
	}
	if rerr := os.Remove(s[0]); rerr != nil {
		return systemError("remove", s[0], rerr)
	}
	return arg.Null
}

// fs.home() is the user's home folder.
func fsHome(args ...object.Object) object.Object {
	if err := arg.Count(args, 0, 0); err != nil {
		return err
	}
	home, herr := os.UserHomeDir()
	if herr != nil {
		return arg.Errorf("can't find the home folder: %v", herr)
	}
	return arg.Str(filepath.ToSlash(home))
}

// fs.join(a, b, ...) puts the parts of a path together with "/", and
// tidies it: fs.join("a/", "./b", "../c") is "a/c".
func fsJoin(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, -1); err != nil {
		return err
	}
	parts := make([]string, len(args))
	for i := range args {
		s, err := arg.String(args, i, "part")
		if err != nil {
			return err
		}
		parts[i] = s
	}
	return arg.Str(filepath.ToSlash(filepath.Join(parts...)))
}

// WriteFileAtomic writes data to a new file next to path and renames it
// over path, so a crash never leaves half a file. A file that was there
// keeps its permissions.
func WriteFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".lizard-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	fail := func(err error) error {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		return fail(err)
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return fail(errors.New("it is a folder"))
		}
		mode = info.Mode().Perm()
	}
	if err := os.Chmod(name, mode); err != nil {
		return fail(err)
	}
	if err := os.Rename(name, path); err != nil {
		return fail(err)
	}
	return nil
}
