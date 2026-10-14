package memory

import (
	"sort"
	"strings"

	"github.com/chenhg5/harness-projects/internal/model"
)

type Match struct {
	Memory  model.Memory `json:"memory"`
	Score   int          `json:"score"`
	Excerpt string       `json:"excerpt"`
}

// Search returns memories containing every query term. Title and tags rank
// above body matches; the files remain the source of truth, not this ranking.
func Search(memories []model.Memory, query string) []Match {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	matches := []Match{}
	if len(terms) == 0 {
		return matches
	}
	for _, item := range memories {
		title := strings.ToLower(item.Title)
		tags := strings.ToLower(strings.Join(item.Tags, " "))
		body := strings.ToLower(item.Content)
		source := strings.ToLower(item.Source)
		score := 0
		for _, term := range terms {
			termScore := 0
			if strings.Contains(title, term) {
				termScore += 4
			}
			if strings.Contains(tags, term) {
				termScore += 3
			}
			if strings.Contains(body, term) {
				termScore++
			}
			if strings.Contains(source, term) {
				termScore++
			}
			if termScore == 0 {
				score = 0
				break
			}
			score += termScore
		}
		if score > 0 {
			matches = append(matches, Match{Memory: item, Score: score, Excerpt: excerpt(item.Content, terms)})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		if matches[i].Memory.UpdatedAt != matches[j].Memory.UpdatedAt {
			return matches[i].Memory.UpdatedAt > matches[j].Memory.UpdatedAt
		}
		return matches[i].Memory.ID < matches[j].Memory.ID
	})
	return matches
}

func excerpt(content string, terms []string) string {
	flat := strings.Join(strings.Fields(content), " ")
	lower := strings.ToLower(flat)
	start := 0
	for _, term := range terms {
		if at := strings.Index(lower, term); at >= 0 {
			start = at - 48
			if start < 0 {
				start = 0
			}
			break
		}
	}
	end := start + 160
	if end > len(flat) {
		end = len(flat)
	}
	return flat[start:end]
}
