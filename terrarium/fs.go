package terrarium

import (
	"errors"
	"io/fs"
	"lizardscript/bundle"
	"lizardscript/evaluator"
	"lizardscript/object"
	"lizardscript/terrarium/internal/arg"
	"os"
	"path/filepath"
	"sort"
)

// fs: files and folders. Paths come back with "/" between their parts on
// every system (Windows takes it too), so a program never has to know which
// one it runs on. In a program built with lizardscript build, the paths
// that start with "bundle:" (os.program_dir()) are its own files, which
// read, exists, list, info, digest and copy read and nothing changes.

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
		"info":   fsInfo,
		"rename": fsRename,
		"copy":   fsCopy,
		"trash":  fsTrash,
		"digest": fsDigest,
	})
}

// systemError is the message of an error from the system, without Go's
// own wording: can't read notes.txt: no such file or directory.
func systemError(verb, path string, err error) *object.Error {
	return arg.Errorf("can't %s %s: %v", verb, path, cause(err))
}

// cause is err without the operation and path Go puts in front.
func cause(err error) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		err = linkErr.Err
	}
	return err
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
	data, rerr := bundle.ReadFile(s[0])
	if rerr != nil {
		return systemError("read", s[0], rerr)
	}
	return arg.Str(string(data))
}

// fs.digest(path, algorithm = "sha256") is the hash of the file's bytes,
// as hex, like text.digest of its text: "sha256" or "fnv". It reads a
// built program's own files (a bundle: path) too.
func fsDigest(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 2); err != nil {
		return err
	}
	path, err := arg.String(args, 0, "path")
	if err != nil {
		return err
	}
	algorithm := evaluator.DigestAlgorithms[0]
	if len(args) == 2 {
		if algorithm, err = arg.String(args, 1, "algorithm"); err != nil {
			return err
		}
	}
	data, rerr := bundle.ReadFile(path)
	if rerr != nil {
		return systemError("read", path, rerr)
	}
	digest, ok := evaluator.Digest(data, algorithm)
	if !ok {
		return evaluator.UnknownDigest(algorithm)
	}
	return arg.Str(digest)
}

// fs.write(path, text) replaces the file with text, or makes it. Another
// program never sees half of it: the text goes to a new file next to it,
// which then takes its place.
func fsWrite(args ...object.Object) object.Object {
	s, err := pathArgs(args, "path", "text")
	if err != nil {
		return err
	}
	if err := writable("write", s[0]); err != nil {
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
	if err := writable("append to", s[0]); err != nil {
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
	_, serr := bundle.Stat(s[0])
	return arg.Bool(serr == nil)
}

// fs.list(dir, options = {}) is the names of what is in the folder, files
// and folders, in alphabetical order. With the option {"info": true} each
// one is a hash instead: its name and what fs.info gives.
func fsList(args ...object.Object) object.Object {
	if err := arg.Count(args, 1, 2); err != nil {
		return err
	}
	dir, err := arg.String(args, 0, "folder")
	if err != nil {
		return err
	}
	withInfo := false
	if len(args) == 2 {
		options, err := arg.Hash(args, 1, "options")
		if err != nil {
			return err
		}
		if withInfo, err = listOptions(options); err != nil {
			return err
		}
	}
	entries, rerr := bundle.ReadDir(dir)
	if rerr != nil {
		return systemError("list", dir, rerr)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	out := &object.Array{Elements: make([]object.Object, 0, len(entries))}
	for _, e := range entries {
		if !withInfo {
			out.Elements = append(out.Elements, arg.Str(e.Name()))
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			// It was removed while the folder was read.
			continue
		}
		out.Elements = append(out.Elements, infoHash(info, e.Name()))
	}
	return out
}

// listOptions reads the options of fs.list: only "info", true or false.
func listOptions(options *object.Hash) (bool, *object.Error) {
	withInfo := false
	var bad *object.Error
	options.Each(func(pair object.HashPair) {
		key := pair.Key.Inspect()
		value, isBool := pair.Value.(*object.Boolean)
		switch {
		case bad != nil:
		case key != "info":
			bad = arg.Errorf("unknown option %s, the only one is info", object.Quote(key))
		case !isBool:
			bad = arg.Errorf("the option info must be true or false, got %s", arg.Name(pair.Value))
		default:
			withInfo = value.Value
		}
	})
	return withInfo, bad
}

// fs.mkdir(path) makes the folder and the ones above it that are missing.
// A folder that is already there is fine.
func fsMkdir(args ...object.Object) object.Object {
	s, err := pathArgs(args, "path")
	if err != nil {
		return err
	}
	if err := writable("make", s[0]); err != nil {
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
	if err := writable("remove", s[0]); err != nil {
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

// fs.info(path) describes a file or folder: {"size": bytes, "mtime":
// seconds since 1970 when it last changed, "dir": whether it is a folder}.
func fsInfo(args ...object.Object) object.Object {
	s, err := pathArgs(args, "path")
	if err != nil {
		return err
	}
	info, serr := bundle.Stat(s[0])
	if serr != nil {
		return systemError("read", s[0], serr)
	}
	return infoHash(info, "")
}

// infoHash is what fs.info gives for info, after the name when it is not "".
func infoHash(info fs.FileInfo, name string) *object.Hash {
	h := object.NewHash()
	if name != "" {
		arg.Set(h, "name", arg.Str(name))
	}
	arg.Set(h, "size", arg.Int64(info.Size()))
	arg.Set(h, "mtime", arg.Int64(info.ModTime().Unix()))
	arg.Set(h, "dir", arg.Bool(info.IsDir()))
	return h
}

// fs.rename(from, to) moves a file or folder to a new path. It refuses to
// replace something that is already at to, so a mistake can't lose a file.
func fsRename(args ...object.Object) object.Object {
	s, err := pathArgs(args, "from", "to")
	if err != nil {
		return err
	}
	for _, p := range s {
		if err := writable("move", p); err != nil {
			return err
		}
	}
	if _, lerr := os.Lstat(s[1]); lerr == nil {
		return arg.Errorf("can't move %s to %s: there is already something there", s[0], s[1])
	}
	if rerr := os.Rename(s[0], s[1]); rerr != nil {
		return systemError("move", s[0], rerr)
	}
	return arg.Null
}

// fs.copy(from, to) copies a file (not a folder), with its permissions. Like
// fs.rename it refuses to replace something at to.
func fsCopy(args ...object.Object) object.Object {
	s, err := pathArgs(args, "from", "to")
	if err != nil {
		return err
	}
	if err := writable("copy to", s[1]); err != nil {
		return err
	}
	if _, lerr := os.Lstat(s[1]); lerr == nil {
		return arg.Errorf("can't copy %s to %s: there is already something there", s[0], s[1])
	}
	info, serr := bundle.Stat(s[0])
	if serr != nil {
		return systemError("copy", s[0], serr)
	}
	if info.IsDir() {
		return arg.Errorf("can't copy %s: it is a folder", s[0])
	}
	data, rerr := bundle.ReadFile(s[0])
	if rerr != nil {
		return systemError("copy", s[0], rerr)
	}
	mode := info.Mode().Perm()
	if bundle.IsPath(s[0]) {
		mode = 0o644
	}
	if werr := os.WriteFile(s[1], data, mode); werr != nil {
		return systemError("copy to", s[1], werr)
	}
	return arg.Null
}

// fs.trash(path) moves a file or folder to the system's Trash (the Recycle
// Bin on Windows), where the user can get it back. It never deletes it: if
// the Trash can't take it, it stays and fs.trash is an error.
func fsTrash(args ...object.Object) object.Object {
	s, err := pathArgs(args, "path")
	if err != nil {
		return err
	}
	if err := writable("move", s[0]); err != nil {
		return err
	}
	if terr := Trash(s[0]); terr != nil {
		return arg.Errorf("can't move %s to the Trash: %v", s[0], cause(terr))
	}
	return arg.Null
}

// writable is the error of a change to a built program's own file, or nil
// for a path on disk.
func writable(verb, path string) *object.Error {
	if bundle.IsPath(path) {
		return systemError(verb, path, bundle.ErrReadOnly)
	}
	return nil
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
