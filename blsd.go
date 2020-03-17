package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/libgit2/git2go"
)

type entry struct {
	path string
	repo *git.Repository
}

var references map[*git.Repository]int

func init() {
	references = make(map[*git.Repository]int)
}

func ignore(path string, repo *git.Repository) bool {
	if repo != nil {
		abs, err := filepath.Abs(path)
		if err != nil {
			return false
		}
		abs, err = filepath.EvalSymlinks(abs)
		if err != nil {
			return false
		}
		base := filepath.Clean(repo.Path() + "..")
		if abs == base {
			return false
		}
		rel, err := filepath.Rel(base, abs)
		if err != nil {
			return false
		}
		ignored, err := repo.IsPathIgnored(rel)
		return err == nil && ignored
	}
	return false
}

func isDir(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode().IsDir()
}

func bfsd(queue []entry, printFiles bool, printDirs bool) []entry {
	newQueue := []entry{}
	for _, e := range queue {
		dir := e.path
		repo := e.repo
		if repo == nil {
			r, err := git.OpenRepository(dir)
			if err == nil {
				repo = r
				references[repo] = 1
			}
		} else {
			references[repo] -= 1
		}
		ignored := ignore(dir, repo)
		if ignored {
			continue
		}

		f, err := os.Open(dir)
		if err != nil {
			continue
		}

		fis, err := f.Readdir(-1)
		if err != nil {
			f.Close()
			continue
		}
		f.Close()

		if dir != "." && printDirs {
			fmt.Println(dir)
		}

		for _, fi := range fis {
			name := fi.Name()
			path := path.Join(dir, name)
			if fi.Mode().IsDir() {
				if repo != nil {
					references[repo] += 1
				}
				newQueue = append(newQueue, entry{path, repo})
			} else if printFiles {
				fmt.Println(path)
			}
		}
		if repo != nil && references[repo] == 1 {
			delete(references, repo)
			repo.Free()
		}
	}
	return newQueue
}

func main() {
	var printFiles bool = true
	var printDirs bool = true
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if arg == "-d" {
				printFiles = false
				printDirs = true
			} else if arg == "-f" {
				printFiles = true
				printDirs = false
			}
		}
	}
	var queue []entry = []entry{entry{".", nil}}
	for len(queue) > 0 {
		queue = bfsd(queue, printFiles, printDirs)
	}
}
