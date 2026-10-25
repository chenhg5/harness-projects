package memory

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FileMatch struct {
	Path    string `json:"path"`
	Score   int    `json:"score"`
	Excerpt string `json:"excerpt"`
}

// SearchFiles searches ordinary Markdown and text notes under context/. The
// structured context/memory tree is excluded because Search already covers it.
func SearchFiles(root, query string) ([]FileMatch, error) {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 {
		return []FileMatch{}, nil
	}
	contextRoot := filepath.Join(root, "context")
	matches := []FileMatch{}
	err := filepath.WalkDir(contextRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != contextRoot && (entry.Name() == "memory" || entry.Name() == "imports") {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == "README.md" || (filepath.Ext(entry.Name()) != ".md" && filepath.Ext(entry.Name()) != ".txt") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		title := strings.ToLower(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		body := strings.ToLower(string(data))
		score := 0
		for _, term := range terms {
			termScore := 0
			if strings.Contains(title, term) {
				termScore += 4
			}
			if strings.Contains(body, term) {
				termScore++
			}
			if termScore == 0 {
				return nil
			}
			score += termScore
		}
		matches = append(matches, FileMatch{Path: filepath.ToSlash(relative), Score: score, Excerpt: excerpt(string(data), terms)})
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return []FileMatch{}, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].Path < matches[j].Path
	})
	return matches, nil
}
