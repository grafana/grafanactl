package config_test

import (
	"testing"

	"github.com/grafana/grafanactl/cmd/config"
	"github.com/grafana/grafanactl/internal/testutils"
)

func Test_CurrentContextCommand(t *testing.T) {
	testCase := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"current-context", "--config", "testdata/config.yaml"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
			testutils.CommandOutputContains("local"),
		},
	}

	testCase.Run(t)
}

func Test_UseContextCommand(t *testing.T) {
	cfg := `current-context: old
contexts:
  old: {}
  new: {}`

	configFile := testutils.CreateTempFile(t, cfg)

	initialConfigTest := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"current-context", "--config", configFile},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
			testutils.CommandOutputContains("old"),
		},
	}
	initialConfigTest.Run(t)

	changeConfigTest := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"use-context", "--config", configFile, "new"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
			testutils.CommandOutputContains("Context set to \"new\""),
		},
	}
	changeConfigTest.Run(t)

	newConfigTest := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"current-context", "--config", configFile},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
			testutils.CommandOutputContains("new"),
		},
	}
	newConfigTest.Run(t)
}

func Test_UseContextCommand_withUnknownContext(t *testing.T) {
	testCase := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"use-context", "--config", "testdata/config.yaml", "unknown-context"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandErrorContains("Context \"unknown-context\" does not exist"),
		},
	}
	testCase.Run(t)
}

func Test_ViewCommand(t *testing.T) {
	testCase := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"view", "--config", "testdata/config.yaml"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
			testutils.CommandOutputContains(`contexts:
  local:
    grafana:
      server: http://localhost:3000/
      token: "**REDACTED**"
  prod:
    grafana:
      server: https://grafana.example.com/
      token: "**REDACTED**"
current-context: local`),
		},
	}

	testCase.Run(t)
}

func Test_ViewCommand_raw(t *testing.T) {
	testCase := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"view", "--config", "testdata/config.yaml", "--raw"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
			testutils.CommandOutputContains(`contexts:
  local:
    grafana:
      server: http://localhost:3000/
      token: local_token
  prod:
    grafana:
      server: https://grafana.example.com/
      token: prod_token
current-context: local`),
		},
	}

	testCase.Run(t)
}

func Test_ViewCommand_minify(t *testing.T) {
	testCase := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"view", "--config", "testdata/config.yaml", "--minify"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
			testutils.CommandOutputContains(`contexts:
  local:
    grafana:
      server: http://localhost:3000/
      token: "**REDACTED**"
current-context: local`),
		},
	}

	testCase.Run(t)
}

func Test_ViewCommand_minify_explicitContext(t *testing.T) {
	testCase := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"view", "--config", "testdata/config.yaml", "--minify", "--context", "prod"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
			testutils.CommandOutputContains(`contexts:
  prod:
    grafana:
      server: https://grafana.example.com/
      token: "**REDACTED**"
current-context: prod`),
		},
	}

	testCase.Run(t)
}

func Test_ViewCommand_outputJson(t *testing.T) {
	testCase := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"view", "--config", "testdata/config.yaml", "-o", "json"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
			testutils.CommandOutputContains(`{
  "contexts": {
    "local": {
      "grafana": {
        "server": "http://localhost:3000/",
        "token": "**REDACTED**"
      }
    },
    "prod": {
      "grafana": {
        "server": "https://grafana.example.com/",
        "token": "**REDACTED**"
      }
    }
  },
  "current-context": "local"
}`),
		},
	}

	testCase.Run(t)
}

func Test_ViewCommand_failsWithNonExistentConfigFile(t *testing.T) {
	testCase := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"view", "--config", "does-not-exist.yaml"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandErrorContains("no such file or directory"),
		},
	}

	testCase.Run(t)
}

func Test_ViewCommand_failsWithUnknownContext(t *testing.T) {
	testCase := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"view", "--config", "testdata/config.yaml", "--context", "unknown-context"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandErrorContains("Context \"unknown-context\" does not exist"),
		},
	}
	testCase.Run(t)
}

func Test_SetCommand(t *testing.T) {
	cfg := `current-context: dev`

	configFile := testutils.CreateTempFile(t, cfg)

	changeConfigTest := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"set", "--config", configFile, "contexts.dev.grafana.server", "https://grafana-dev.example"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
		},
	}
	changeConfigTest.Run(t)

	viewCmd := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"view", "--config", configFile, "--minify"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
			testutils.CommandOutputContains(`contexts:
  dev:
    grafana:
      server: https://grafana-dev.example
current-context: dev`),
		},
	}
	viewCmd.Run(t)
}

func Test_UnsetCommand(t *testing.T) {
	cfg := `contexts:
  dev:
    grafana:
      server: https://grafana-dev.example
      user: remove-me-please
current-context: dev`

	configFile := testutils.CreateTempFile(t, cfg)

	changeConfigTest := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"unset", "--config", configFile, "contexts.dev.grafana.user"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
		},
	}
	changeConfigTest.Run(t)

	viewCmd := testutils.CommandTestCase{
		Cmd:     config.Command(testutils.NullLogger()),
		Command: []string{"view", "--config", configFile, "--minify"},
		Assertions: []testutils.CommandAssertion{
			testutils.CommandSuccess(),
			testutils.CommandOutputContains(`contexts:
  dev:
    grafana:
      server: https://grafana-dev.example
current-context: dev`),
		},
	}
	viewCmd.Run(t)
}
