package conditions

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
	"github.com/kruglovmax/stack/pkg/types"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// When func
func When(stack types.Stack, condition string) (result bool) {
	if condition == "" {
		result = true
		return
	}
	result = checkCondition(stack, condition, stack.GetView().(map[string]interface{}))
	return
}

// Wait func
func Wait(stack types.Stack, condition string, timeout time.Duration) (result bool) {
	if condition == "" {
		result = true
		return
	}
	log.Logger.Info().
		Str("condition", condition).
		Str("in stack", stack.GetWorkdir()).
		Msg("Waiting for")
	waitLoopDone := make(chan int)
	go waitLoop(stack, condition, waitLoopDone)
	select {
	case <-waitLoopDone:
		result = true
	case <-time.After(timeout):
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Str("timeout", fmt.Sprintf("%s", timeout)).
			Str("in stack", stack.GetWorkdir()).
			Str("condition", condition).
			Msg("Waiting failed")
	}

	return
}

func waitLoop(stack types.Stack, condition string, exit chan int) {
	for {
		if checkCondition(stack, condition, stack.GetView().(map[string]interface{})) {
			break
		}
		time.Sleep(100 * time.Millisecond)
		log.Logger.Trace().
			Str("condition", condition).Msg("Waiting for")
	}
	exit <- 0
}

func checkCondition(stack types.Stack, condition string, varsMap map[string]interface{}) (result bool) {
	var declarations []*exprpb.Decl
	for key := range varsMap {
		declarations = append(declarations, decls.NewVar(key, decls.Any))
	}
	envOption := cel.Declarations(declarations...)
	env, err := cel.NewEnv(envOption)
	misc.CheckIfErr(err)
	ast, iss := env.Compile(condition)
	if iss.Err() != nil {
		log.Logger.Error().
			Str("condition", condition).
			Str("in stack", stack.GetWorkdir()).
			Send()
		log.Logger.Debug().
			Msg(string(debug.Stack()))
	}
	if iss.Err() != nil {
		log.Logger.Debug().
			Str("condition", condition).
			Str("in stack", stack.GetWorkdir()).
			Msgf("Error %s\n", iss.Err().Error())
		return
	}
	prg, err := env.Program(ast)
	misc.CheckIfErr(err)
	out, _, err := prg.Eval(varsMap)
	if err != nil {
		return
	}
	value, ok := out.Value().(bool)
	if !ok {
		log.Logger.Warn().
			Str("result type", fmt.Sprintf("%T", out.Value())).
			Str("result value", spew.Sprint(out.Value())).
			Str("type expected", "bool").
			Str("in stack", stack.GetWorkdir()).
			Str("condition", condition).
			Send()
	}
	result = value && ok

	return
}
