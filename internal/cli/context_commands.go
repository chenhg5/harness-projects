package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chenhg5/harness-projects/internal/fileops"
	hpmemory "github.com/chenhg5/harness-projects/internal/memory"
	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/store"
)

type contextMatch struct {
	Type    string        `json:"type"`
	Memory  *model.Memory `json:"memory,omitempty"`
	Path    string        `json:"path,omitempty"`
	Score   int           `json:"score"`
	Excerpt string        `json:"excerpt"`
}

func (a app) contextAdd(args []string) int {
	fs := newFlags("context add")
	projectPath := fs.String("project", "", "project directory")
	id := fs.String("id", "", "memory id")
	kind := fs.String("kind", "", "decision|lesson|preference|research")
	title := fs.String("title", "", "memory title")
	source := fs.String("from", "", "content file or - for stdin")
	tags := fs.String("tags", "", "comma-separated tags")
	reference := fs.String("source", "", "source reference")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" || *id == "" || *kind == "" || *title == "" || *source == "" {
		return a.fail(failure{2, "invalid_arguments", "--project, --id, --kind, --title, and --from are required", "Run hp context add --help", ""})
	}
	content, err := fileops.ReadSource(*source, a.in)
	if err != nil {
		return a.operationError(err, *source)
	}
	memoryTags, err := commaList(*tags)
	if err != nil {
		return a.parseError(err)
	}
	now := timestamp()
	item := model.Memory{SchemaVersion: model.ResourceSchemaVersion, ID: *id, Kind: *kind, Title: *title, Content: string(content), Tags: memoryTags, Source: *reference, CreatedAt: now, UpdatedAt: now}
	if err := model.ValidateMemory(item); err != nil {
		return a.operationError(err, *id)
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if _, err := s.Memory(*kind, *id); err == nil {
		return a.fail(failure{5, "conflict", "memory already exists: " + *id, "Choose another id", *id})
	} else if !store.IsNotFound(err) {
		return a.operationError(err, *id)
	}
	result := map[string]any{"action": "context.add", "memory": item}
	if *dryRun {
		a.emit(result, "Would add memory "+item.ID, "json")
		return 10
	}
	if err := s.SaveMemory(item, false); err != nil {
		return a.operationError(err, *id)
	}
	a.emit(result, "Memory added: "+item.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) contextList(args []string) int {
	fs := newFlags("context list")
	projectPath := fs.String("project", "", "project directory")
	kind := fs.String("kind", "", "filter by kind")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.parseError(fmt.Errorf("--project is required"))
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	items, err := s.Memories(*kind)
	if err != nil {
		return a.operationError(err, *kind)
	}
	a.emit(items, fmt.Sprintf("%d memory item(s)", len(items)), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) contextGet(args []string) int {
	fs := newFlags("context get")
	projectPath := fs.String("project", "", "project directory")
	kind := fs.String("kind", "", "memory kind")
	id := fs.String("id", "", "memory id")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" || *kind == "" || *id == "" {
		return a.parseError(fmt.Errorf("--project, --kind, and --id are required"))
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	item, err := s.Memory(*kind, *id)
	if err != nil {
		return a.operationError(err, *id)
	}
	a.emit(item, item.Title, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) contextSearch(args []string) int {
	fs := newFlags("context search")
	projectPath := fs.String("project", "", "project directory")
	query := fs.String("query", "", "search terms")
	kind := fs.String("kind", "", "filter by kind")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" || strings.TrimSpace(*query) == "" {
		return a.parseError(fmt.Errorf("--project and --query are required"))
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	items, err := s.Memories(*kind)
	if err != nil {
		return a.operationError(err, *kind)
	}
	matches := []contextMatch{}
	for _, match := range hpmemory.Search(items, *query) {
		memory := match.Memory
		matches = append(matches, contextMatch{Type: "memory", Memory: &memory, Score: match.Score, Excerpt: match.Excerpt})
	}
	fileMatches, err := hpmemory.SearchFiles(s.Root, *query)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	for _, match := range fileMatches {
		matches = append(matches, contextMatch{Type: "file", Path: match.Path, Score: match.Score, Excerpt: match.Excerpt})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].Path < matches[j].Path
	})
	a.emit(matches, fmt.Sprintf("%d match(es)", len(matches)), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}
