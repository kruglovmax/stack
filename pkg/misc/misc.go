package misc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kruglovmax/stack/pkg/app"
	"github.com/kruglovmax/stack/pkg/consts"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/types"
	sopsDecrypt "go.mozilla.org/sops/v3/decrypt"
	"sigs.k8s.io/yaml"
)

// LoadYAML func
func LoadYAML(yamlInput string, result interface{}) {
	var yamlErr error
	yamlErr = yaml.Unmarshal([]byte(yamlInput), &result)
	if yamlErr != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(yamlInput))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(yamlErr.Error())
	}
}

// LoadYAMLFromFile func
func LoadYAMLFromFile(fileName string, result interface{}) {
	content, ioErr := ioutil.ReadFile(fileName)
	if ioErr != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(fileName))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(ioErr.Error())
	}
	LoadYAML(string(content), &result)
}

// LoadYAMLFromSopsFile func
func LoadYAMLFromSopsFile(fileName string, result interface{}) {
	content, err := sopsDecrypt.File(fileName, "yaml")
	if err != nil {
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msgf("[ SOPS ] File %s decryption Error. Check internet connection.\n"+err.Error(), fileName)
	}
	LoadYAML(string(content), &result)
}

// ToYAML func
func ToYAML(object interface{}) string {
	y, err := yaml.Marshal(object)
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(object))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("YAML marshal error\n" + err.Error())
	}
	return string(y)
}

// ToInterface func
func ToInterface(object interface{}) (result interface{}) {
	LoadYAML(ToJSON(object), &result)
	return
}

// ToJSON func
func ToJSON(object interface{}) string {
	y, err := json.Marshal(object)
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(object))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("JSON marshal error\n" + err.Error())
	}
	return string(y)
}

// UniqueStr func
func UniqueStr(input []string) (output []string) {
	output = make([]string, 0, len(input))
	keys := make(map[string]bool)
	for _, entry := range input {
		if _, ok := keys[entry]; !ok {
			keys[entry] = true
			output = append(output, entry)
		}
	}
	return
}

// WaitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

// FindPath func
func FindPath(dir string, searchPaths ...string) (output string, err error) {
	if filepath.IsAbs(dir) && PathIsDir(dir) {
		output, _ = filepath.Abs(dir)
		err = nil
		return
	}
	for _, searchPath := range searchPaths {
		var files []os.FileInfo
		files, err = ioutil.ReadDir(searchPath)
		if err != nil {
			return
		}
		for _, fileItem := range files {
			if filepath.Base(fileItem.Name()) == dir && fileItem.IsDir() {
				output = filepath.Join(searchPath, fileItem.Name())
				return
			}
		}
	}
	err = fmt.Errorf(consts.MessagePathNotFoundInSearchPaths, dir, spew.Sdump(searchPaths))
	return
}

// CheckIfErr func
func CheckIfErr(err error, stacks ...types.Stack) {
	if err != nil {
		for _, stack := range stacks {
			PrintStackTrace(stack)
		}
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(err.Error())
	}
}

// GitClone func
func GitClone(parentWG *sync.WaitGroup, gitClonePath, gitURL, gitRef string, fetchIfExists bool, noWaitForOthers bool) {
	if parentWG != nil {
		defer parentWG.Done()
	}

	var err error
	if !noWaitForOthers {
		app.App.Mutex.GitWorkMutex.Lock()
		defer app.App.Mutex.GitWorkMutex.Unlock()
	}
	os.MkdirAll(gitClonePath, os.ModePerm)
	var gitRepo *git.Repository
	gitRepo, err = git.PlainClone(gitClonePath, false, &git.CloneOptions{
		URL:      gitURL,
		Progress: nil,
	})
	if err == git.ErrRepositoryAlreadyExists {
		// fetch repo
		if fetchIfExists {
			gitRepo, err = git.PlainOpen(gitClonePath)
			if err != nil {
				return
			}
			err = gitRepo.Fetch(&git.FetchOptions{Progress: os.Stderr})
		} else {
			return
		}
	}
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return
	}
	var gitWorkTree *git.Worktree
	gitWorkTree, err = gitRepo.Worktree()
	if err != nil {
		CheckIfErr(err)
	}
	err = gitWorkTree.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(gitRef),
	})
	return
}

// GetRunItemOutputType func
// func GetRunItemOutputType(item interface{}) []interface{} {
// 	result, ok := (item).(map[string]interface{})["output"].([]interface{})
// 	if ok {
// 		return result
// 	}
// 	return nil
// }

// GetObjectValue func
func GetObjectValue(object interface{}, key string) interface{} {
	switch object.(type) {
	case map[string]interface{}:
		return object.(map[string]interface{})[key]
	case []interface{}:
		i, err := strconv.Atoi(key)
		if err != nil {
			log.Logger.Trace().
				Msg(spew.Sdump(key))
			log.Logger.Debug().
				Msg(string(debug.Stack()))
			log.Logger.Fatal().Str("Index", key).
				Msg(err.Error())
		}
		return object.([]interface{})[i]
	default:
		log.Logger.Warn().
			Msg("Bad value\n" + string(debug.Stack()))
		return nil
	}
}

// GetObject func
func GetObject(root interface{}, path string) interface{} {
	splitResult := strings.SplitN(strings.TrimLeft(path, "."), ".", 2)
	left := splitResult[0]
	if len(splitResult) > 0 && left != "" {
		if len(splitResult) > 1 {
			right := splitResult[1]
			return GetObject(GetObjectValue(root, left), right)
		}
		return GetObjectValue(root, left)
	}
	return root
}

// GetDirsByRegexp func
func GetDirsByRegexp(pwd string, pattern string) (dirs []string) {
	// ^(?:monitoring|(.*))$
	pattern = "^(" + pattern + ")$"
	files, err := ioutil.ReadDir(pwd)
	if err != nil {
		dirs = nil
		return
	}
	for _, fileItem := range files {
		if fileItem.IsDir() {
			re, err := regexp.Compile(pattern)
			if err != nil {
				log.Logger.Fatal().
					Msg(err.Error() + "\n" + string(debug.Stack()))
			}
			matches := re.FindStringSubmatch(fileItem.Name())
			if len(matches) > 0 && matches[1] != "" {
				dirs = append(dirs, fileItem.Name())
			}
		}
	}
	return
}

// ReadFileFromPath func
func ReadFileFromPath(path string) (output string) {
	loadTemplateFromWalkPath := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if PathIsExists(path) && !info.IsDir() {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			output = output + string(content) + "\n"
		}
		return nil
	}

	fullpath, err := filepath.Abs(path)
	if PathIsExists(fullpath) && (err == nil) {
		err := filepath.Walk(fullpath, loadTemplateFromWalkPath)
		CheckIfErr(err)
		return output
	}
	err = fmt.Errorf("Path is not exists: %s", fullpath)
	CheckIfErr(err)
	return ""
}

// PathIsExists returns whether the given file or directory exists
func PathIsExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}

// PathIsDir func
func PathIsDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		return true
	default:
		return false
	}
}

// PathIsFile func
func PathIsFile(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	switch mode := fi.Mode(); {
	case mode.IsRegular():
		return true
	default:
		return false
	}
}

// PrintStackTrace func
func PrintStackTrace(stack types.Stack) {
	log.Logger.Error().
		Str("stack", stack.GetWorkdir()).
		Msg("From")
	parentStack := stack.GetParent()
	for parentStack != nil {
		log.Logger.Error().
			Str("stack", parentStack.GetWorkdir()).
			Msg("Parent")
		parentStack = parentStack.GetParent()
	}
}

// FindStackFileInDir func
func FindStackFileInDir(dir string) (stackFile string) {
	dirBase := filepath.Base(dir)
	switch {
	case PathIsExists(filepath.Join(dir, consts.StackDefaultFileName+".yaml")):
		stackFile = filepath.Clean(filepath.Join(dir, consts.StackDefaultFileName+".yaml"))
	case PathIsExists(filepath.Join(dir, consts.StackDefaultFileName+".yml")):
		stackFile = filepath.Clean(filepath.Join(dir, consts.StackDefaultFileName+".yml"))
	case PathIsExists(filepath.Join(dir, consts.StackDefaultFileName+".json")):
		stackFile = filepath.Clean(filepath.Join(dir, consts.StackDefaultFileName+".json"))
	case PathIsExists(filepath.Join(dir, dirBase+".yaml")):
		stackFile = filepath.Clean(filepath.Join(dir, dirBase+".yaml"))
	case PathIsExists(filepath.Join(dir, dirBase+".yml")):
		stackFile = filepath.Clean(filepath.Join(dir, dirBase+".yml"))
	case PathIsExists(filepath.Join(dir, dirBase+".json")):
		stackFile = filepath.Clean(filepath.Join(dir, dirBase+".json"))
	default:
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Str("workdir", dir).
			Msg("Stack file is not found in")
	}
	return
}

// GetDirName func
func GetDirName(fullpath string) (dirName string) {
	dirName = filepath.Base(filepath.Dir(fullpath))
	return
}

// GetDirPath func
func GetDirPath(fullpath string) (dirPath string) {
	dirPath = filepath.Dir(fullpath)
	return
}

// GetStackPathRelativeToTheRootStack func
func GetStackPathRelativeToTheRootStack(stack types.Stack) (output string) {
	var err error
	output, err = filepath.Rel(*app.App.Config.Workdir, stack.GetWorkdir())
	CheckIfErr(err, stack)
	return
}

// KeyIsExist func
func KeyIsExist(str string, stringMap map[string]interface{}) bool {
	for b := range stringMap {
		if b == str {
			return true
		}
	}
	return false
}
