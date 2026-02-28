package builtins

import (
	"encoding/json"
	"fmt"
	"strings"

	logql "github.com/grafana/loki/v3/pkg/logql/syntax"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/types"
)

func ValidateLogQL() func(*rego.Rego) {
	return rego.Function2(
		&rego.Function{
			Name: "validate_logql",
			Decl: types.NewFunction(
				types.Args(
					// the PromQL expression
					types.S,
					// Dashboard variables
					types.Any{},
				),
				// empty string for valid expression, en error message otherwise
				types.S,
			),
		},
		func(_ rego.BuiltinContext, exprTerm *ast.Term, dashboardVarsTerm *ast.Term) (*ast.Term, error) {
			// TODO: variables should be expanded (Grafana builtin ones + the ones defined by the dashboard)
			expr, ok := exprTerm.Value.(ast.String)
			// The function is undefined for non-string inputs.
			if !ok {
				return nil, nil
			}

			dashboardVars := make([]dashboardVariable, 0)
			if err := json.Unmarshal([]byte(dashboardVarsTerm.String()), &dashboardVars); err != nil {
				return nil, err
			}

			if _, err := parseLogQL(string(expr), dashboardVars); err != nil {
				return ast.StringTerm(err.Error()), nil
			}

			return ast.StringTerm(""), nil
		},
	)
}

//nolint:ireturn
func parseLogQL(expr string, variables []dashboardVariable) (logql.Expr, error) {
	expr, err := expandLogQLVariables(expr, variables)
	if err != nil {
		return nil, fmt.Errorf("could not expand variables: %w", err)
	}
	return logql.ParseExpr(expr)
}

func expandLogQLVariables(expr string, variables []dashboardVariable) (string, error) {
	lines := strings.Split(expr, "\n")
	for i, line := range lines {
		parts := strings.Split(line, "\"")
		for j, part := range parts {
			if j%2 == 1 {
				// Inside a double quote string, just add it
				continue
			}

			// Accumulator to store the processed submatches
			var subparts []string
			// Cursor indicates where we are in the part being processed
			cursor := 0
			for _, v := range variableRegexp.FindAllStringSubmatchIndex(part, -1) {
				// Add all until match starts
				subparts = append(subparts, part[cursor:v[0]])
				// Iterate on all the subgroups and find the one that matched
				for k := 2; k < len(v); k += 2 {
					if v[k] < 0 {
						continue
					}
					// Replace the match with sample value
					val, err := variableSampleValue(part[v[k]:v[k+1]], variables)
					if err != nil {
						return "", err
					}
					// If the variable is within square brackets, remove the '$' prefix
					if strings.HasPrefix(part[v[0]-1:v[0]], "[") && strings.HasSuffix(part[v[1]:v[1]+1], "]") {
						val = strings.TrimPrefix(val, "$")
					}
					subparts = append(subparts, val)
				}
				// Move the start cursor at the end of the current match
				cursor = v[1]
			}
			// Add rest of the string
			subparts = append(subparts, part[cursor:])
			// Merge all back into the parts
			parts[j] = strings.Join(subparts, "")
		}
		lines[i] = strings.Join(parts, "\"")
	}
	result := strings.Join(lines, "\n")
	return result, nil
}
