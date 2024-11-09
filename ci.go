//go:build ci

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
)

const dist = "/tmp/tt"

func main() {
	args := givenArgs()
	for _, target := range args {
		switch target {
		case "setup":
			cmd := "github.com/gregoryv/gocolor/cmd/gocolor@latest"
			sh("go", "install", cmd)

		case "build":
			sh("go", "build", "-o", dist, "./cmd/tt")

		case "test":
			sh("go", "test", "./...")

		case "dist":
			os.MkdirAll(dist, 0755)
			f := filepath.Join(dist, "release_notes.txt")
			os.WriteFile(f, releaseNotes(), 0644)

		case "clear":
			os.RemoveAll(dist)

		case "release-notes":
			fmt.Println(string(releaseNotes()))

		default:
			fmt.Fprint(os.Stderr, "unknown target:", target)
			os.Exit(1)
		}
	}
}

func givenArgs() []string {
	args := slices.Clone(os.Args[1:])
	if len(args) == 0 {
		args = append(args, "build", "test")
	}
	return args
}

func sh(app string, args ...string) {
	c := exec.Command(app, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		os.Exit(1)
	}
}

// reelaseNotes returns top changelog section
func releaseNotes() []byte {
	h2 := []byte("## [")
	from := bytes.Index(changelog, h2)
	to := bytes.Index(changelog[from+len(h2):], h2) + len(h2)
	notes := changelog[from:]
	if to > 0 {
		notes = changelog[from : from+to]
	}
	return bytes.TrimSpace(notes)
}

//go:embed changelog.md
var changelog []byte
