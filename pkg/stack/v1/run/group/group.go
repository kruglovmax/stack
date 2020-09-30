package group

import (
	"fmt"
	"sync"
	"time"

	"github.com/kruglovmax/stack/pkg/app"
	"github.com/kruglovmax/stack/pkg/conditions"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
	"github.com/kruglovmax/stack/pkg/types"
)

// groupItem type
type groupItem struct {
	Group       []types.RunItem `json:"group,omitempty"`
	Parallel    bool            `json:"parallel,omitempty"`
	When        string          `json:"when,omitempty"`
	Wait        string          `json:"wait,omitempty"`
	RunTimeout  time.Duration   `json:"runTimeout,omitempty"`
	WaitTimeout time.Duration   `json:"waitTimeout,omitempty"`
}

// Exec func
func (item *groupItem) Exec(parentWG *sync.WaitGroup, stack types.Stack) {
	if parentWG != nil {
		defer parentWG.Done()
	}
	if !conditions.When(stack, item.When) {
		return
	}
	if !conditions.Wait(stack, item.Wait, item.WaitTimeout) {
		return
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go item.execGroup(&wg, stack)

	if misc.WaitTimeout(&wg, item.RunTimeout) {
		log.Logger.Fatal().
			Str("stack", stack.GetWorkdir()).
			Str("timeout", fmt.Sprint(item.RunTimeout)).
			Msg("Group waiting failed")
	}
}

func (item *groupItem) execGroup(parentWG *sync.WaitGroup, stack types.Stack) {
	defer parentWG.Done()
	if item.Parallel {
		var wg sync.WaitGroup
		for _, runItem := range item.Group {
			wg.Add(1)
			go runItem.Exec(&wg, stack)
		}
		wg.Wait()
	} else {
		for _, runItem := range item.Group {
			var wg sync.WaitGroup
			wg.Add(1)
			go runItem.Exec(&wg, stack)
			wg.Wait()
		}
	}
}

// Parse func
func Parse(stack types.Stack, item map[string]interface{}) types.RunItem {
	output := new(groupItem)
	output.Group = stack.GetRunItemsParser().ParseRun(stack, item["group"].([]interface{}))
	parallel := item["parallel"]
	if parallel == nil {
		parallel = false
	}
	output.Parallel = parallel.(bool)
	whenCondition := (item)["when"]
	waitCondition := (item)["wait"]
	if whenCondition != nil {
		output.When = whenCondition.(string)
	}
	if waitCondition != nil {
		output.Wait = waitCondition.(string)
	}
	var err error
	runTimeout := item["runTimeout"]
	output.RunTimeout = *app.App.Config.DefaultTimeout
	if runTimeout != nil {
		output.RunTimeout, err = time.ParseDuration(runTimeout.(string))
		misc.CheckIfErr(err)
	}
	waitTimeout := item["waitTimeout"]
	output.WaitTimeout = *app.App.Config.DefaultTimeout
	if waitTimeout != nil {
		output.WaitTimeout, err = time.ParseDuration(waitTimeout.(string))
		misc.CheckIfErr(err)
	}

	return output
}
