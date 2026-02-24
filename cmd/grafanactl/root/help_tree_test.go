package root

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractArgs(t *testing.T) {
	tests := []struct {
		use  string
		want string
	}{
		{"get", ""},
		{"get UID", "UID"},
		{"push [SELECTOR]...", "[SELECTOR]..."},
		{"set KEY VALUE", "KEY VALUE"},
		{"delete [RESOURCE]... [flags]", "[RESOURCE]... [flags]"},
	}

	for _, tt := range tests {
		t.Run(tt.use, func(t *testing.T) {
			assert.Equal(t, tt.want, extractArgs(tt.use))
		})
	}
}

func TestDetectEnum(t *testing.T) {
	tests := []struct {
		name  string
		usage string
		want  string
	}{
		{
			name:  "simple pipe-separated enum",
			usage: "Error handling strategy: ignore|fail|abort",
			want:  "ignore|fail|abort",
		},
		{
			name:  "two values",
			usage: "Output format: json|yaml",
			want:  "json|yaml",
		},
		{
			name:  "no enum",
			usage: "The name of the resource to get",
			want:  "",
		},
		{
			name:  "single word is not enum",
			usage: "Only one value here",
			want:  "",
		},
		{
			name: "multi-line dash enum",
			usage: `Error handling strategy:
  ignore — skip errors
  fail   — stop on first error
  abort  — abort everything`,
			want: "ignore|fail|abort",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, detectEnum(tt.usage))
		})
	}
}

func TestFormatFlag(t *testing.T) {
	tests := []struct {
		name      string
		flag      pflag.Flag
		want      string
	}{
		{
			name: "bool flag",
			flag: makeFlag("dry-run", "", "bool", "", "Run without making changes"),
			want: "[--dry-run]",
		},
		{
			name: "bool with shorthand",
			flag: makeFlag("verbose", "v", "bool", "", "Verbose output"),
			want: "[-v|--verbose]",
		},
		{
			name: "count flag",
			flag: makeFlag("verbose", "v", "count", "", "Verbosity level"),
			want: "[-v|--verbose count]",
		},
		{
			name: "string with default",
			flag: makeFlag("output", "o", "string", "json", "Output format"),
			want: "[-o|--output json]",
		},
		{
			name: "string without default",
			flag: makeFlag("name", "", "string", "", "Resource name"),
			want: "[--name <name>]",
		},
		{
			name: "string with enum usage",
			flag: makeFlag("on-error", "", "string", "", "Strategy: ignore|fail|abort"),
			want: "[--on-error ignore|fail|abort]",
		},
		{
			name: "int with default",
			flag: makeFlag("port", "", "int", "8080", "Server port"),
			want: "[--port int=8080]",
		},
		{
			name: "int without default",
			flag: makeFlag("count", "", "int", "0", "Number of items"),
			want: "[--count int]",
		},
		{
			name: "stringSlice empty",
			flag: makeFlag("path", "p", "stringSlice", "[]", "Paths"),
			want: "[-p|--path ...]",
		},
		{
			name: "stringSlice with default",
			flag: makeFlag("path", "p", "stringSlice", "[./resources]", "Paths"),
			want: "[-p|--path ./resources]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatFlag(&tt.flag))
		})
	}
}

func TestFormatFlags_SkipsHiddenAndHelp(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Bool("visible", false, "A visible flag")
	flags.Bool("help", false, "Show help")
	flags.Bool("secret", false, "A hidden flag")
	require.NoError(t, flags.MarkHidden("secret"))

	result := formatFlags(flags, nil)
	assert.Contains(t, result, "--visible")
	assert.NotContains(t, result, "--help")
	assert.NotContains(t, result, "--secret")
}

func TestFormatFlags_SkipsInherited(t *testing.T) {
	inherited := pflag.NewFlagSet("inherited", pflag.ContinueOnError)
	inherited.Bool("global", false, "A global flag")

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Bool("global", false, "A global flag")
	flags.Bool("local", false, "A local flag")

	result := formatFlags(flags, inherited)
	assert.NotContains(t, result, "--global")
	assert.Contains(t, result, "--local")
}

func TestFormatFlags_NilFlagSet(t *testing.T) {
	assert.Equal(t, "", formatFlags(nil, nil))
}

func TestDisplayName(t *testing.T) {
	t.Run("with annotation", func(t *testing.T) {
		cmd := &cobra.Command{
			Use: "myapp",
			Annotations: map[string]string{
				cobra.CommandDisplayNameAnnotation: "my-app",
			},
		}
		assert.Equal(t, "my-app", displayName(cmd))
	})

	t.Run("without annotation", func(t *testing.T) {
		cmd := &cobra.Command{Use: "myapp"}
		assert.Equal(t, "myapp", displayName(cmd))
	})
}

func TestHasSubcommands(t *testing.T) {
	t.Run("no children", func(t *testing.T) {
		cmd := &cobra.Command{Use: "leaf"}
		assert.False(t, hasSubcommands(cmd))
	})

	t.Run("only help child", func(t *testing.T) {
		cmd := &cobra.Command{Use: "parent"}
		cmd.AddCommand(&cobra.Command{Use: "help"})
		assert.False(t, hasSubcommands(cmd))
	})

	t.Run("only hidden child", func(t *testing.T) {
		cmd := &cobra.Command{Use: "parent"}
		cmd.AddCommand(&cobra.Command{Use: "internal", Hidden: true})
		assert.False(t, hasSubcommands(cmd))
	})

	t.Run("visible child", func(t *testing.T) {
		cmd := &cobra.Command{Use: "parent"}
		cmd.AddCommand(&cobra.Command{Use: "sub", Short: "A subcommand"})
		assert.True(t, hasSubcommands(cmd))
	})
}

func TestPrintFullTree(t *testing.T) {
	root := buildTestTree()

	var buf bytes.Buffer
	err := printFullTree(&buf, root)
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Root line with global flags.
	assert.Contains(t, lines[0], "testctl")
	assert.Contains(t, lines[0], "[--no-color]")

	// Group command with description.
	assert.Contains(t, output, "things")
	assert.Contains(t, output, "# Manage things")

	// Leaf commands indented under group.
	assert.Contains(t, output, "  get")
	assert.Contains(t, output, "  list")

	// Hidden commands excluded.
	assert.NotContains(t, output, "hidden-cmd")

	// help and help-tree excluded.
	assert.NotContains(t, output, "  help")
	assert.NotContains(t, output, "  help-tree")
}

func TestPrintSubtree(t *testing.T) {
	root := buildTestTree()
	things, _, err := root.Find([]string{"things"})
	require.NoError(t, err)

	var buf bytes.Buffer
	err = printSubtree(&buf, root, things)
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Root line shows full path with inherited flags.
	assert.Contains(t, lines[0], "testctl things")
	assert.Contains(t, lines[0], "# Manage things")

	// Children listed below.
	assert.Contains(t, output, "get")
	assert.Contains(t, output, "list")
}

func TestPrintSubtree_LeafCommand(t *testing.T) {
	root := buildTestTree()
	get, _, err := root.Find([]string{"things", "get"})
	require.NoError(t, err)

	var buf bytes.Buffer
	err = printSubtree(&buf, root, get)
	require.NoError(t, err)

	output := buf.String()

	// Shows command path with args and local flags.
	assert.Contains(t, output, "testctl get")
	assert.Contains(t, output, "NAME")
}

func TestPrintChildren_SortedAlphabetically(t *testing.T) {
	parent := &cobra.Command{Use: "parent"}
	parent.AddCommand(&cobra.Command{Use: "zebra"})
	parent.AddCommand(&cobra.Command{Use: "alpha"})
	parent.AddCommand(&cobra.Command{Use: "middle"})

	var buf bytes.Buffer
	err := printChildren(&buf, parent, 0, pflag.NewFlagSet("empty", pflag.ContinueOnError))
	require.NoError(t, err)

	output := buf.String()
	alphaIdx := strings.Index(output, "alpha")
	middleIdx := strings.Index(output, "middle")
	zebraIdx := strings.Index(output, "zebra")

	assert.Less(t, alphaIdx, middleIdx)
	assert.Less(t, middleIdx, zebraIdx)
}

func TestHelpTreeCmd_FullTree(t *testing.T) {
	root := buildTestTree()
	root.AddCommand(helpTreeCmd())

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"help-tree"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "testctl")
	assert.Contains(t, output, "things")
	assert.NotContains(t, output, "help-tree")
}

func TestHelpTreeCmd_Subtree(t *testing.T) {
	root := buildTestTree()
	root.AddCommand(helpTreeCmd())

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"help-tree", "things"})

	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "testctl things")
	assert.Contains(t, output, "get")
}

func TestHelpTreeCmd_UnknownCommand(t *testing.T) {
	root := buildTestTree()
	root.AddCommand(helpTreeCmd())

	root.SetArgs([]string{"help-tree", "nonexistent"})

	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestCollectAllInherited(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().Bool("global", false, "Global flag")

	child := &cobra.Command{Use: "child"}
	child.PersistentFlags().Bool("child-persistent", false, "Child persistent")
	root.AddCommand(child)

	// Need to initialize inherited flags by calling InheritedFlags.
	result := collectAllInherited(child)

	assert.NotNil(t, result.Lookup("global"))
	assert.NotNil(t, result.Lookup("child-persistent"))
}

// buildTestTree creates a minimal command tree for testing.
func buildTestTree() *cobra.Command {
	root := &cobra.Command{
		Use: "testctl",
		Annotations: map[string]string{
			cobra.CommandDisplayNameAnnotation: "testctl",
		},
	}
	root.PersistentFlags().Bool("no-color", false, "Disable colors")

	things := &cobra.Command{Use: "things", Short: "Manage things"}

	get := &cobra.Command{
		Use:   "get NAME",
		Short: "Get a thing",
		RunE:  func(_ *cobra.Command, _ []string) error { return nil },
	}
	get.Flags().StringP("output", "o", "json", "Output format")

	list := &cobra.Command{
		Use:   "list",
		Short: "List things",
		RunE:  func(_ *cobra.Command, _ []string) error { return nil },
	}
	list.Flags().StringP("output", "o", "text", "Output format")

	things.AddCommand(get, list)

	hidden := &cobra.Command{
		Use:    "hidden-cmd",
		Hidden: true,
		RunE:   func(_ *cobra.Command, _ []string) error { return nil },
	}

	root.AddCommand(things, hidden)

	return root
}

// makeFlag creates a pflag.Flag with the given properties for testing formatFlag.
func makeFlag(name, shorthand, typeName, defValue, usage string) pflag.Flag {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	switch typeName {
	case "bool":
		fs.BoolP(name, shorthand, false, usage)
	case "count":
		fs.CountP(name, shorthand, usage)
	case "string":
		fs.StringP(name, shorthand, defValue, usage)
		return *fs.Lookup(name)
	case "int":
		fs.IntP(name, shorthand, 0, usage)
		if defValue != "" && defValue != "0" {
			// Set the DefValue string directly to match what real flags look like.
			fs.Lookup(name).DefValue = defValue
		}
	case "stringSlice":
		if defValue == "[]" || defValue == "" {
			fs.StringSliceP(name, shorthand, nil, usage)
		} else {
			trimmed := strings.Trim(defValue, "[]")
			fs.StringSliceP(name, shorthand, []string{trimmed}, usage)
		}
		return *fs.Lookup(name)
	default:
		fs.StringP(name, shorthand, defValue, usage)
		return *fs.Lookup(name)
	}

	return *fs.Lookup(name)
}
