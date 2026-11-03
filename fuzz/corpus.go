package fuzz

import (
	"os"
	"path/filepath"
	"strings"
)

// root is the folder of the repo, seen from this one.
const root = ".."

// Corpus gives the programs the fuzzers start from: every .liz file of
// tests/ and examples/, and every code block of the guide in docs/learn
// (the ones fenced with a bare ```, which are lizard-script; ```text and
// ```sh blocks are output and shell).
func Corpus() []string {
	var programs []string
	for _, pattern := range []string{
		"tests/valid/*.liz", "tests/valid/*/*.liz", "tests/errors/*.liz",
		"examples/*.liz", "examples/*/*.liz",
	} {
		files, _ := filepath.Glob(filepath.Join(root, pattern))
		for _, file := range files {
			if source, err := os.ReadFile(file); err == nil {
				programs = append(programs, strings.ReplaceAll(string(source), "\r\n", "\n"))
			}
		}
	}
	guide, _ := filepath.Glob(filepath.Join(root, "docs", "learn", "*", "*.md"))
	for _, file := range guide {
		if source, err := os.ReadFile(file); err == nil {
			programs = append(programs, codeBlocks(string(source))...)
		}
	}
	return programs
}

// codeBlocks gives the bodies of the blocks fenced with a bare ``` in a
// Markdown text.
func codeBlocks(markdown string) []string {
	var blocks []string
	var current []string
	inside, code := false, false
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "```") {
			if inside && code {
				current = append(current, line)
			}
			continue
		}
		if inside {
			if code && len(current) > 0 {
				blocks = append(blocks, strings.Join(current, "\n"))
			}
			inside, current = false, nil
			continue
		}
		inside, code = true, trimmed == "```"
	}
	return blocks
}
