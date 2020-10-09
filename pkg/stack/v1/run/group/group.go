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

	rawItem map[string]interface{}
	stack   types.Stack
}

// New func
func New(stack types.Stack, rawItem map[string]interface{}) types.RunItem {
	item := new(groupItem)
	item.rawItem = rawItem
	item.stack = stack

	return item
}

// Exec func
func (item *groupItem) Exec(parentWG *sync.WaitGroup, stack types.Stack) {
	item.parse()
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

	if item.RunTimeout != 0 {
		if misc.WaitTimeout(&wg, item.RunTimeout) {
			log.Logger.Fatal().
				Str("stack", stack.GetWorkdir()).
				Str("timeout", fmt.Sprint(item.RunTimeout)).
				Msg("Group waiting failed")
		}
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

func (item *groupItem) parse() {
	item.Group = item.stack.GetRunItemsParser().ParseRun(item.stack, item.rawItem["group"].([]interface{}))
	parallel := item.rawItem["parallel"]
	if parallel == nil {
		parallel = false
	}
	item.Parallel = parallel.(bool)
	whenCondition := (item.rawItem)["when"]
	waitCondition := (item.rawItem)["wait"]
	if whenCondition != nil {
		item.When = whenCondition.(string)
	}
	if waitCondition != nil {
		item.Wait = waitCondition.(string)
	}
	var err error
	runTimeout := item.rawItem["runTimeout"]
	item.RunTimeout = 0
	if runTimeout != nil {
		item.RunTimeout, err = time.ParseDuration(runTimeout.(string))
		misc.CheckIfErr(err)
	}
	waitTimeout := item.rawItem["waitTimeout"]
	item.WaitTimeout = *app.App.Config.DefaultTimeout
	if waitTimeout != nil {
		item.WaitTimeout, err = time.ParseDuration(waitTimeout.(string))
		misc.CheckIfErr(err)
	}
}
