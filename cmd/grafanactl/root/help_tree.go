package root

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func helpTreeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "help-tree [command]",
		Short: "Print a compact command tree for LLM and scripting use",
		Long:  "Print a compact, token-efficient representation of the full command tree (or a subtree) showing all commands, arguments, and flags.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			if len(args) > 0 {
				target, _, err := root.Find(strings.Split(args[0], " "))
				if err != nil {
					return fmt.Errorf("unknown command %q", args[0])
				}
				return printSubtree(cmd.OutOrStdout(), root, target)
			}

			return printFullTree(cmd.OutOrStdout(), root)
		},
	}
}

func printFullTree(w io.Writer, root *cobra.Command) error {
	globalFlags := formatFlags(root.PersistentFlags(), nil)
	fmt.Fprintf(w, "%s %s\n", displayName(root), globalFlags)

	return printChildren(w, root, 1, root.PersistentFlags())
}

func printSubtree(w io.Writer, root *cobra.Command, target *cobra.Command) error {
	// Build the command path from root to target.
	path := displayName(root)
	if target != root {
		path += " " + target.Name()
	}

	inheritedFlags := formatFlags(target.InheritedFlags(), nil)
	line := path + " " + inheritedFlags
	if hasSubcommands(target) {
		line += "  # " + target.Short
	} else {
		args := extractArgs(target.Use)
		localFlags := formatFlags(target.LocalFlags(), target.InheritedFlags())
		line = path + " " + args + " " + localFlags
	}

	fmt.Fprintln(w, strings.TrimRight(line, " "))

	return printChildren(w, target, 1, collectAllInherited(target))
}

func printChildren(w io.Writer, parent *cobra.Command, depth int, inherited *pflag.FlagSet) error {
	cmds := parent.Commands()
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].Name() < cmds[j].Name()
	})

	indent := strings.Repeat("  ", depth)

	for _, child := range cmds {
		if child.Hidden || child.Name() == "help" || child.Name() == "help-tree" {
			continue
		}

		if hasSubcommands(child) {
			fmt.Fprintf(w, "%s%-30s  # %s\n", indent, child.Name(), child.Short)

			if err := printChildren(w, child, depth+1, inherited); err != nil {
				return err
			}
		} else {
			args := extractArgs(child.Use)
			localFlags := formatFlags(child.LocalFlags(), inherited)

			parts := []string{child.Name()}
			if args != "" {
				parts = append(parts, args)
			}
			if localFlags != "" {
				parts = append(parts, localFlags)
			}

			fmt.Fprintf(w, "%s%s\n", indent, strings.Join(parts, " "))
		}
	}

	return nil
}

func displayName(cmd *cobra.Command) string {
	if name, ok := cmd.Annotations[cobra.CommandDisplayNameAnnotation]; ok {
		return name
	}

	return cmd.Name()
}

func extractArgs(use string) string {
	// Cobra Use field is "command-name [args...]" — strip the command name.
	parts := strings.SplitN(use, " ", 2)
	if len(parts) < 2 {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func formatFlags(flags *pflag.FlagSet, inherited *pflag.FlagSet) string {
	if flags == nil {
		return ""
	}

	var parts []string

	flags.VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Name == "help" {
			return
		}

		// Skip flags that are in the inherited set.
		if inherited != nil {
			if inh := inherited.Lookup(f.Name); inh != nil {
				return
			}
		}

		parts = append(parts, formatFlag(f))
	})

	return strings.Join(parts, " ")
}

func formatFlag(f *pflag.Flag) string {
	name := "--" + f.Name
	if f.Shorthand != "" {
		name = "-" + f.Shorthand + "|" + name
	}

	typeName := f.Value.Type()

	switch typeName {
	case "bool":
		return "[" + name + "]"

	case "count":
		return "[" + name + " count]"

	case "string":
		// Try to detect enum values from usage text.
		if enums := detectEnum(f.Usage); enums != "" {
			return "[" + name + " " + enums + "]"
		}

		if f.DefValue != "" && f.DefValue != "\"\"" {
			return "[" + name + " " + f.DefValue + "]"
		}

		return "[" + name + " <" + f.Name + ">]"

	case "strings", "stringSlice", "stringArray":
		def := strings.Trim(f.DefValue, "[]")
		if def != "" {
			return "[" + name + " " + def + "]"
		}

		return "[" + name + " ...]"

	case "int":
		if f.DefValue != "" && f.DefValue != "0" {
			return "[" + name + " int=" + f.DefValue + "]"
		}

		return "[" + name + " int]"

	default:
		if f.DefValue != "" {
			return "[" + name + " " + typeName + "=" + f.DefValue + "]"
		}

		return "[" + name + " " + typeName + "]"
	}
}

func detectEnum(usage string) string {
	// Look for patterns like "ignore|fail|abort" or lines with em-dash separated options.
	// First, try simple "word|word|word" pattern.
	simpleEnum := regexp.MustCompile(`\b([a-zA-Z][\w-]*(?:\|[a-zA-Z][\w-]*)+)\b`)
	if m := simpleEnum.FindString(usage); m != "" {
		return m
	}

	// Look for multi-line enum patterns:
	//   ignore — description
	//   fail   — description
	//   abort  — description
	dashEnum := regexp.MustCompile(`(?m)^\s+(\w+)\s+[—–-]`)
	matches := dashEnum.FindAllStringSubmatch(usage, -1)

	if len(matches) >= 2 {
		var vals []string
		for _, m := range matches {
			vals = append(vals, m[1])
		}

		return strings.Join(vals, "|")
	}

	return ""
}

func hasSubcommands(cmd *cobra.Command) bool {
	for _, child := range cmd.Commands() {
		if !child.Hidden && child.Name() != "help" && child.Name() != "help-tree" {
			return true
		}
	}

	return false
}

func collectAllInherited(cmd *cobra.Command) *pflag.FlagSet {
	result := pflag.NewFlagSet("inherited", pflag.ContinueOnError)

	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		result.AddFlag(f)
	})

	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		result.AddFlag(f)
	})

	return result
}
