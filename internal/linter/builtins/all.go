package builtins

import (
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/tester"
)

func All() []func(*rego.Rego) {
	return []func(*rego.Rego){
		ValidateLogQL(),
		ValidatePromql(),
	}
}

func Tester() []*tester.Builtin {
	return []*tester.Builtin{
		{
			Decl: &ast.Builtin{
				Name: validatePromqlMeta.Name,
				Decl: validatePromqlMeta.Decl,
			},
			Func: ValidatePromql(),
		},
		{
			Decl: &ast.Builtin{
				Name: validateLogQLMeta.Name,
				Decl: validateLogQLMeta.Decl,
			},
			Func: ValidateLogQL(),
		},
	}
}
