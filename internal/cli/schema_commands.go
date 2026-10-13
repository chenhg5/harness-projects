package cli

import (
	"sort"
	"strings"
)

func (a app) schemaList(args []string) int {
	fs := newFlags("schema list")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	commands := make([]string, 0, len(commandSchemas))
	for command := range commandSchemas {
		commands = append(commands, command)
	}
	sort.Strings(commands)
	a.emit(map[string]any{"commands": commands}, strings.Join(commands, "\n"), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) schemaGet(args []string) int {
	fs := newFlags("schema get")
	command := fs.String("command", "", "command name, e.g. project init (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	schema, ok := commandSchemas[*command]
	if !ok {
		return a.fail(failure{3, "not_found", "unknown command schema: " + *command, "Run hp schema list", *command})
	}
	a.emit(schema, commandHelp[*command], selectedFormat(*format, *jsonFlag, a.out))
	return 0
}
