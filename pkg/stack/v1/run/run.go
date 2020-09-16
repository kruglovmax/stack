package run

import (
	"os"

	"github.com/kruglovmax/stack/pkg/app"
	"github.com/kruglovmax/stack/pkg/stack/v1/run/gomplate"
	"github.com/kruglovmax/stack/pkg/stack/v1/run/pongo2"
	"github.com/kruglovmax/stack/pkg/stack/v1/run/script"
	"github.com/kruglovmax/stack/pkg/types"
)

// ParseRun func
func ParseRun(input []interface{}, workdir string) (output []types.RunItem) {
	output = make([]types.RunItem, 0, len(input))
	for _, item := range input {
		runItem := parseRunItem(item, workdir)
		if runItem != nil {
			output = append(output, runItem)
		}
	}
	return
}

func parseRunItem(item interface{}, workdir string) (output types.RunItem) {
	app.App.Mutex.CurrentWorkDirMutex.Lock()
	defer app.App.Mutex.CurrentWorkDirMutex.Unlock()
	os.Chdir(workdir)

	switch item.(type) {
	case map[string]interface{}:
		switch {
		case item.(map[string]interface{})["gomplate"] != nil:
			output = gomplate.Parse(workdir, item)
		case item.(map[string]interface{})["pongo2"] != nil:
			output = pongo2.Parse(workdir, item)
		case item.(map[string]interface{})["script"] != nil:
			output = script.Parse(workdir, item)
		}
	}
	return
}
