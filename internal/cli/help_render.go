package cli

import (
	"fmt"
	"strings"
)

func renderCommandHelp(schema commandSchema) string {
	enriched := enrichCommandSchema(schema)
	var builder strings.Builder

	builder.WriteString("Usage:\n  hp ")
	builder.WriteString(schema.Command)
	for _, flag := range enriched.Flags {
		argument := renderFlagArgument(flag)
		if flag.Required {
			builder.WriteString(" ")
			builder.WriteString(argument)
		} else {
			builder.WriteString(" [")
			builder.WriteString(argument)
			builder.WriteString("]")
		}
	}
	builder.WriteString("\n\nExamples:\n")
	for _, example := range schema.Examples {
		builder.WriteString("  ")
		builder.WriteString(example)
		builder.WriteString("\n")
	}
	builder.WriteString("\nDescription:\n  ")
	builder.WriteString(schema.Description)
	builder.WriteString(".\n\nOutput:\n  ")
	builder.WriteString(strings.Join(enriched.Output.Formats, " or "))
	builder.WriteString("; default: ")
	builder.WriteString(enriched.Output.Default)
	builder.WriteString(".")
	if enriched.Output.JSONShape != "" {
		builder.WriteString(" ")
		builder.WriteString(enriched.Output.JSONShape)
	}
	if enriched.Output.DryRun {
		builder.WriteString(" Successful dry runs emit JSON and exit 10.")
	}
	builder.WriteString("\n  Errors are JSON on stderr; inspect error, exitCode, retryable, and suggestion.\n\nOptions:\n")
	for _, flag := range enriched.Flags {
		builder.WriteString("  ")
		builder.WriteString(renderFlagArgument(flag))
		builder.WriteString("\n      ")
		builder.WriteString(flag.Description)
		if flag.Required {
			builder.WriteString(" Required.")
		} else {
			builder.WriteString(" Default: ")
			builder.WriteString(renderDefault(flag.Default))
			builder.WriteString(".")
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func renderFlagArgument(flag flagSchemaOutput) string {
	prefix := "--" + flag.Name
	if flag.Type == "boolean" {
		return prefix
	}
	placeholder := flag.Type
	if len(flag.Values) > 0 {
		placeholder = strings.Join(flag.Values, "|")
	}
	return fmt.Sprintf("%s <%s>", prefix, placeholder)
}

func renderDefault(value any) string {
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return "empty"
		}
		return fmt.Sprintf("%q", typed)
	default:
		return fmt.Sprint(value)
	}
}
