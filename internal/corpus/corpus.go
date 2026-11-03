// Package corpus lists the lizard-script programs of this repository: the
// .liz files and the code blocks of the guide. Tests of the tools that read
// programs (the formatter, the checker) run over all of them.
package corpus

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Program is one program of the repository.
type Program struct {
	Name   string // its path from the root, and for a block of the guide ":line" of its first line
	Source string
	Block  bool // a code block of the guide, not a file
}

// Root is the folder of the repository.
func Root() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// Files are the .liz files under each of dirs, folders of the root, in
// order. A folder that isn't there has none.
func Files(dirs ...string) []Program {
	var programs []Program
	for _, dir := range dirs {
		filepath.WalkDir(filepath.Join(Root(), dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".liz") {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			programs = append(programs, Program{Name: name(path), Source: string(source)})
			return nil
		})
	}
	return programs
}

// GuideBlocks are the code blocks of the guide (docs/learn) written in
// lizard-script: the ones whose fence has no language, as the guide marks
// its code (```text is output, ```sh a shell).
func GuideBlocks() []Program {
	var programs []Program
	paths, _ := filepath.Glob(filepath.Join(Root(), "docs", "learn", "*", "*.md"))
	for _, path := range paths {
		text, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		programs = append(programs, blocks(name(path), string(text))...)
	}
	return programs
}

// blocks are the lizard-script blocks of the markdown text of the file
// called file.
func blocks(file, text string) []Program {
	var programs []Program
	var code []string
	start, inside, lizard := 0, false, false
	for i, line := range strings.Split(text, "\n") {
		fence := strings.TrimSpace(line)
		switch {
		case !inside && strings.HasPrefix(fence, "```"):
			inside, lizard, start, code = true, fence == "```", i+2, nil
		case inside && fence == "```":
			inside = false
			if lizard {
				programs = append(programs, Program{
					Name:   file + ":" + strconv.Itoa(start),
					Source: strings.Join(code, "\n") + "\n",
					Block:  true,
				})
			}
		case inside:
			code = append(code, line)
		}
	}
	return programs
}

func name(path string) string {
	rel, err := filepath.Rel(Root(), path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
