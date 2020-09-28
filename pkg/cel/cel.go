package cel

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// ComputeCEL func
func ComputeCEL(expression string, varsMap map[string]interface{}) (result interface{}, err error) {
	var declarations []*exprpb.Decl
	var env *cel.Env
	var prg cel.Program
	var out ref.Val

	for key := range varsMap {
		declarations = append(declarations, decls.NewVar(key, decls.Any))
	}
	envOption := cel.Declarations(declarations...)
	env, err = cel.NewEnv(envOption)
	if err != nil {
		return
	}
	ast, iss := env.Compile(expression)
	if iss.Err() != nil {
		err = iss.Err()
		return
	}
	prg, err = env.Program(ast)
	if err != nil {
		return
	}
	out, _, err = prg.Eval(varsMap)
	if err != nil {
		return
	}
	result = out.Value()

	return
}
