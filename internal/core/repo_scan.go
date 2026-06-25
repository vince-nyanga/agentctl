package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type ScanResult struct {
	Name   string
	Path   string
	Remote string
}

func ScanRepos(root string) ([]ScanResult, error) {
	found := map[string]ScanResult{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if name == ".git" || name == "node_modules" || name == "vendor" || name == ".agentctl" {
			return filepath.SkipDir
		}
		if hasGitMarker(path) && IsGitRepo(path) {
			top, err := GitTopLevel(path)
			if err != nil || top == "" {
				top = path
			}
			found[top] = ScanResult{Name: filepath.Base(top), Path: top, Remote: GitRemote(top)}
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	results := make([]ScanResult, 0, len(found))
	for _, result := range found {
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	return results, nil
}

func hasGitMarker(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}
