package gitclone

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/flytam/filenamify"
	"github.com/kruglovmax/stack/pkg/app"
	"github.com/kruglovmax/stack/pkg/conditions"
	"github.com/kruglovmax/stack/pkg/consts"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
	"github.com/kruglovmax/stack/pkg/types"
)

// gitcloneItem type
type gitcloneItem struct {
	Repo        string        `json:"gitclone,omitempty"`
	Ref         string        `json:"ref,omitempty"`
	Dir         string        `json:"dir,omitempty"`
	When        string        `json:"when,omitempty"`
	Wait        string        `json:"wait,omitempty"`
	RunTimeout  time.Duration `json:"runTimeout,omitempty"`
	WaitTimeout time.Duration `json:"waitTimeout,omitempty"`

	rawItem map[string]interface{}
	stack   types.Stack
}

// New func
func New(stack types.Stack, rawItem map[string]interface{}) types.RunItem {
	item := new(gitcloneItem)
	item.rawItem = rawItem
	item.stack = stack

	return item
}

// Exec func
func (item *gitcloneItem) Exec(parentWG *sync.WaitGroup) {
	item.parse()
	if parentWG != nil {
		defer parentWG.Done()
	}
	if !conditions.When(item.stack, item.When) {
		return
	}
	if !conditions.Wait(item.stack, item.Wait, item.WaitTimeout) {
		return
	}

	gitcloneSubDir, err := filenamify.Filenamify(item.Repo, filenamify.Options{Replacement: "_"})
	misc.CheckIfErr(err, item.stack)

	var wg sync.WaitGroup
	wg.Add(1)
	dir := item.Dir
	if dir == "" {
		dir = filepath.Join(*app.App.Config.Workdir, consts.GitCloneDir, gitcloneSubDir, item.Ref)
	}
	go misc.GitClone(&wg, dir, item.Repo, item.Ref, true, true)
	if misc.WaitTimeout(&wg, item.RunTimeout) {
		log.Logger.Fatal().
			Str("stack", item.stack.GetWorkdir()).
			Str("timeout", fmt.Sprint(item.RunTimeout)).
			Msg("Git clone waiting failed")
	}
}

func (item *gitcloneItem) parse() {
	item.Repo = item.rawItem["gitclone"].(string)
	ref, ok := item.rawItem["ref"].(string)
	if !ok || ref == "" {
		ref = "master"
	}
	item.Ref = ref
	whenCondition := item.rawItem["when"]
	waitCondition := item.rawItem["wait"]
	if whenCondition != nil {
		item.When = whenCondition.(string)
	}
	if waitCondition != nil {
		item.Wait = waitCondition.(string)
	}
	var err error
	runTimeout := item.rawItem["runTimeout"]
	item.RunTimeout = *app.App.Config.DefaultTimeout
	if runTimeout != nil {
		item.RunTimeout, err = time.ParseDuration(runTimeout.(string))
		misc.CheckIfErr(err, item.stack)
	}
	waitTimeout := item.rawItem["waitTimeout"]
	item.WaitTimeout = *app.App.Config.DefaultTimeout
	if waitTimeout != nil {
		item.WaitTimeout, err = time.ParseDuration(waitTimeout.(string))
		misc.CheckIfErr(err, item.stack)
	}

	if _, ok := item.rawItem["dir"]; ok {
		app.App.Mutex.CurrentWorkDirMutex.Lock()
		os.Chdir(item.stack.GetWorkdir())
		item.Dir, err = filepath.Abs(item.rawItem["dir"].(string))
		app.App.Mutex.CurrentWorkDirMutex.Unlock()
		misc.CheckIfErr(err, item.stack)
	}
}
