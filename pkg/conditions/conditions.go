package conditions

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	celgo "github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	celtypes "github.com/google/cel-go/common/types"
	celref "github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter/functions"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"

	"github.com/kruglovmax/stack/pkg/app"
	"github.com/kruglovmax/stack/pkg/cel"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/types"
)

const sleepTime = 100 * time.Millisecond

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

// WaitGroupAdd func
func WaitGroupAdd(stack types.Stack, waitgroup string) *sync.WaitGroup {
	wgKey := waitgroup

	stackMap := stack.GetView().(map[string]interface{})
	stackMap["stack"] = stackMap
	computed, err := cel.ComputeCEL(waitgroup, stackMap)
	if _, ok := computed.(string); err == nil && ok {
		wgKey = computed.(string)
	}
	wg, ok := app.App.WaitGroups[wgKey]
	if !ok {
		app.App.WaitGroups[wgKey] = new(sync.WaitGroup)
		wg = app.App.WaitGroups[wgKey]
	}
	wg.Add(1)
	return wg
}

func waitLoop(stack types.Stack, condition string, exit chan int) {
	for {
		if checkCondition(stack, condition, stack.GetView().(map[string]interface{})) {
			break
		}
		time.Sleep(sleepTime)
		log.Logger.Trace().
			Str("condition", condition).Msg("Waiting for")
	}
	exit <- 0
}

func checkCondition(stack types.Stack, condition string, varsMap map[string]interface{}) (result bool) {
	var celAddon cel.CELaddons
	waitGroupFunc := &functions.Overload{
		Operator: "waitGroup_string",
		Unary: func(lhs celref.Val) celref.Val {
			wg, ok := app.App.WaitGroups[fmt.Sprint(lhs)]
			if ok {
				log.Logger.Trace().
					Str("condition", condition).Msg("Waiting for")
				wg.Wait()
				return celtypes.True
			}
			return celtypes.False
		}}
	celAddon.Decls = append(celAddon.Decls, decls.NewFunction("waitGroup",
		decls.NewOverload("waitGroup_string",
			[]*exprpb.Type{decls.String},
			decls.Bool)))
	celAddon.ProgramOption = append(celAddon.ProgramOption, celgo.Functions(waitGroupFunc))

	stackMap := stack.GetView().(map[string]interface{})
	stackMap["stack"] = stackMap
	computed, err := cel.ComputeCEL(condition, stackMap, celAddon)

	if err != nil {
		log.Logger.Debug().
			Str("condition", condition).
			Str("in stack", stack.GetWorkdir()).
			Msgf("Error %s\n", err.Error())
		return
	}
	value, ok := computed.(bool)

	if !ok {
		log.Logger.Warn().
			Str("result type", fmt.Sprintf("%T", computed)).
			Str("result value", spew.Sprint(computed)).
			Str("type expected", "bool").
			Str("in stack", stack.GetWorkdir()).
			Str("condition", condition).
			Send()
	}
	result = value && ok

	return
}
