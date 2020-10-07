package libs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"

	"github.com/flytam/filenamify"
	"github.com/kruglovmax/stack/pkg/app"
	"github.com/kruglovmax/stack/pkg/consts"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
)

// errors
var (
	ErrBadLibItem error = errors.New(consts.MessageLibsBadItem)
)

// ParseAndInitLibs func
func ParseAndInitLibs(input []interface{}, workdir string) (output []string) {
	output = make([]string, 0, len(input)+1)
	output = append(output, workdir)
	for _, item := range input {
		libPath, err := parseLibsItem(item, workdir)
		misc.CheckIfErr(err)
		output = append(output, libPath)
	}
	output = append(output, *app.App.Config.Workdir)
	output = misc.UniqueStr(output)
	return
}

func parseLibsItem(input interface{}, workdir string) (libPath string, err error) {
	log.Logger.Debug().
		Msgf(consts.MessageLibsParseAndInit, input)
	switch input.(type) {
	case string:
		libPath, err = misc.FindPath(input.(string), workdir, *app.App.Config.Workdir)
		return
	case map[string]interface{}:
		libItem := input.(map[string]interface{})
		switch {
		case libItem["git"] != nil:
			var output string
			output, err = filenamify.Filenamify(libItem["git"].(string), filenamify.Options{Replacement: "_"})
			if err != nil {
				log.Logger.Fatal().
					Msg(err.Error() + "\n" + string(debug.Stack()))
			}
			app.App.Mutex.CurrentWorkDirMutex.Lock()
			defer app.App.Mutex.CurrentWorkDirMutex.Unlock()
			os.Chdir(*app.App.Config.Workdir)
			gitURL, _ := libItem["git"].(string)
			gitRef, ok := libItem["ref"].(string)
			if !ok {
				gitRef = "HEAD"
			}
			gitPath, ok := libItem["path"].(string)
			if !ok {
				gitPath = "."
			}
			var gitClonePath string
			gitClonePath, err = filepath.Abs(filepath.Join(*app.App.Config.GitLibsPath, output, gitRef))
			if err != nil {
				return
			}

			var wg sync.WaitGroup
			wg.Add(1)
			misc.GitClone(&wg, gitClonePath, gitURL, gitRef, false, false)
			misc.WaitTimeout(&wg, *app.App.Config.DefaultTimeout)

			libPath = filepath.Clean(filepath.Join(gitClonePath, gitPath))
			if !misc.PathIsDir(libPath) {
				err = fmt.Errorf(consts.MessageLibsGitBadPathInRepo, gitPath, gitURL)
				return
			}
			return
		}
	}
	err = ErrBadLibItem
	return
}
