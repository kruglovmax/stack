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
}

// Exec func
func (item *gitcloneItem) Exec(parentWG *sync.WaitGroup, stack types.Stack) {
	if parentWG != nil {
		defer parentWG.Done()
	}
	if !conditions.Wait(stack, item.Wait, item.WaitTimeout) {
		return
	}
	if !conditions.When(stack, item.When) {
		return
	}

	gitcloneSubDir, err := filenamify.Filenamify(item.Repo, filenamify.Options{Replacement: "_"})
	misc.CheckIfErr(err)

	var wg sync.WaitGroup
	wg.Add(1)
	dir := item.Dir
	if dir == "" {
		dir = filepath.Join(*app.App.Config.Workdir, consts.GitCloneDir, gitcloneSubDir, item.Ref)
	}
	go misc.GitClone(&wg, dir, item.Repo, item.Ref)
	if misc.WaitTimeout(&wg, item.RunTimeout) {
		log.Logger.Fatal().
			Str("stack", stack.GetWorkdir()).
			Str("timeout", fmt.Sprint(item.RunTimeout)).
			Msg("Git clone waiting failed")
	}
}

// Parse func
func Parse(stack types.Stack, item map[string]interface{}) types.RunItem {
	output := new(gitcloneItem)
	output.Repo = item["gitclone"].(string)
	ref, ok := item["ref"].(string)
	if !ok || ref == "" {
		ref = "master"
	}
	output.Ref = ref
	whenCondition := item["when"]
	waitCondition := item["wait"]
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

	if _, ok := item["dir"]; ok {
		app.App.Mutex.CurrentWorkDirMutex.Lock()
		os.Chdir(stack.GetWorkdir())
		output.Dir, err = filepath.Abs(item["dir"].(string))
		app.App.Mutex.CurrentWorkDirMutex.Unlock()
		misc.CheckIfErr(err)
	}

	return output
}
