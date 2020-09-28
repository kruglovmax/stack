package jsonnet

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Jeffail/gabs/v2"
	jsonnet "github.com/google/go-jsonnet"
	"github.com/imdario/mergo"
	"github.com/joeycumines/go-dotnotation/dotnotation"
	"github.com/kruglovmax/stack/pkg/app"
	"github.com/kruglovmax/stack/pkg/cel"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
	"github.com/kruglovmax/stack/pkg/types"
	"gopkg.in/yaml.v2"
)

// jsonnetItem type
type jsonnetItem struct {
	Jsonnet     string        `json:"jsonnet,omitempty"`
	Paths       []string      `json:"paths,omitempty"`
	Vars        interface{}   `json:"vars,omitempty"`
	Output      []interface{} `json:"output,omitempty"`
	When        string        `json:"when,omitempty"`
	Wait        string        `json:"wait,omitempty"`
	RunTimeout  time.Duration `json:"runTimeout,omitempty"`
	WaitTimeout time.Duration `json:"waitTimeout,omitempty"`
}

// Exec func
func (item *jsonnetItem) Exec(parentWG *sync.WaitGroup, stack types.Stack) {
	if parentWG != nil {
		defer parentWG.Done()
	}
	if !cel.Wait(stack, item.Wait, item.WaitTimeout) {
		return
	}
	if !cel.When(stack, item.When) {
		return
	}

	app.App.Mutex.CurrentWorkDirMutex.Lock()
	os.Chdir(stack.GetWorkdir())
	var parsedString, jsonnetSnippet string
	switch {
	case item.Jsonnet != "":
		jsonnetSnippet = item.Jsonnet
	case item.Paths != nil:
		switch {
		case len(item.Paths) == 1:
			path := item.Paths[0]
			switch {
			case misc.PathIsFile(path):
				dir := filepath.Dir(path)
				file := filepath.Base(path)
				os.Chdir(dir)
				var err error
				var content []byte
				content, err = ioutil.ReadFile(file)
				misc.CheckIfErr(err)
				jsonnetSnippet = string(content)
			case misc.PathIsDir(path):
				os.Chdir(path)
				// TODO
			}
		case len(item.Paths) > 1:
			// TODO
		}
	}

	switch item.Vars.(type) {
	case map[string]interface{}:
		var wg sync.WaitGroup
		wg.Add(1)
		parsedString = processJsonnet(&wg, item.Vars.(map[string]interface{}), jsonnetSnippet)
		if misc.WaitTimeout(&wg, item.RunTimeout) {
			log.Logger.Fatal().
				Str("stack", stack.GetWorkdir()).
				Str("timeout", fmt.Sprint(item.RunTimeout)).
				Msg("Jsonnet waiting failed")
		}
	case string:
		vars, err := dotnotation.Get(stack.GetView(), item.Vars.(string))
		misc.CheckIfErr(err)
		var wg sync.WaitGroup
		wg.Add(1)
		parsedString = processJsonnet(&wg, vars, jsonnetSnippet)
		if misc.WaitTimeout(&wg, item.RunTimeout) {
			log.Logger.Fatal().
				Str("stack", stack.GetWorkdir()).
				Str("timeout", fmt.Sprint(item.RunTimeout)).
				Msg("Jsonnet waiting failed")
		}
	case nil:
		var wg sync.WaitGroup
		wg.Add(1)
		parsedString = processJsonnet(&wg, stack.GetView(), jsonnetSnippet)
		if misc.WaitTimeout(&wg, item.RunTimeout) {
			log.Logger.Fatal().
				Str("stack", stack.GetWorkdir()).
				Str("timeout", fmt.Sprint(item.RunTimeout)).
				Msg("Jsonnet waiting failed")
		}
	default:
		err := fmt.Errorf("Unable to parse run item. Bad vars key")
		misc.CheckIfErr(err)
	}

	app.App.Mutex.CurrentWorkDirMutex.Unlock()

	if item.Output == nil {
		return
	}

	for _, v := range item.Output {
		switch v.(type) {
		case string:
			switch v.(string) {
			case "stdout":
				messagesChannel, listenerChannel := app.App.StdOut.StartOutputForObject()
				app.App.StdOut.SendStringForObject(messagesChannel, parsedString)
				app.App.StdOut.FinishOutputForObject(messagesChannel, listenerChannel)
			case "stderr":
				messagesChannel, listenerChannel := app.App.StdErr.StartOutputForObject()
				app.App.StdErr.SendStringForObject(messagesChannel, parsedString)
				app.App.StdErr.FinishOutputForObject(messagesChannel, listenerChannel)
			}
		case map[string]interface{}:
			if v.(map[string]interface{})["yml2var"] != nil {
				var value map[string]interface{}
				yml2var := v.(map[string]interface{})["yml2var"].(string)
				err := yaml.Unmarshal([]byte(parsedString), &value)
				misc.CheckIfErr(err)
				switch {
				case strings.HasPrefix(yml2var, "vars"):
					key := strings.TrimPrefix(strings.TrimPrefix(yml2var, "vars"), ".")
					setVar := gabs.New()
					if key == "" {
						setVar.Set(value)
					} else {
						setVar.SetP(value, key)
					}
					stack.AddRawVarsRight(setVar.Data().(map[string]interface{}))
				case strings.HasPrefix(yml2var, "flags"):
					key := strings.TrimPrefix(strings.TrimPrefix(yml2var, "flags"), ".")
					setVar := gabs.New()
					if key == "" {
						setVar.Set(value)
					} else {
						setVar.SetP(value, key)
					}
					stack.GetFlags().Mux.Lock()
					mergo.Merge(&stack.GetFlags().Vars, setVar.Data().(map[string]interface{}), mergo.WithOverwriteWithEmptyValue)
					stack.GetFlags().Mux.Unlock()
				case strings.HasPrefix(yml2var, "locals"):
					key := strings.TrimPrefix(strings.TrimPrefix(yml2var, "locals"), ".")
					setVar := gabs.New()
					if key == "" {
						setVar.Set(value)
					} else {
						setVar.SetP(value, key)
					}
					stack.GetLocals().Mux.Lock()
					mergo.Merge(&stack.GetLocals().Vars, setVar.Data().(map[string]interface{}), mergo.WithOverwriteWithEmptyValue)
					stack.GetLocals().Mux.Unlock()
				default:
					log.Logger.Fatal().
						Str("yml2var", yml2var).
						Str("in stack", stack.GetWorkdir()).
						Msg("Bad output var")
				}
			}
			if v.(map[string]interface{})["str2var"] != nil {
				str2var := v.(map[string]interface{})["str2var"].(string)
				switch {
				case strings.HasPrefix(str2var, "vars."):
					key := strings.TrimPrefix(str2var, "vars.")
					setVar := gabs.New()
					setVar.SetP(parsedString, key)
					stack.AddRawVarsRight(setVar.Data().(map[string]interface{}))
				case strings.HasPrefix(str2var, "flags."):
					key := strings.TrimPrefix(str2var, "flags.")
					setVar := gabs.New()
					setVar.SetP(parsedString, key)
					stack.GetFlags().Mux.Lock()
					mergo.Merge(&stack.GetFlags().Vars, setVar.Data().(map[string]interface{}), mergo.WithOverwriteWithEmptyValue)
					stack.GetFlags().Mux.Unlock()
				case strings.HasPrefix(str2var, "locals."):
					key := strings.TrimPrefix(str2var, "locals.")
					setVar := gabs.New()
					setVar.SetP(parsedString, key)
					stack.GetLocals().Mux.Lock()
					mergo.Merge(&stack.GetLocals().Vars, setVar.Data().(map[string]interface{}), mergo.WithOverwriteWithEmptyValue)
					stack.GetLocals().Mux.Unlock()
				default:
					log.Logger.Fatal().
						Str("str2var", str2var).
						Str("in stack", stack.GetWorkdir()).
						Msg("Bad output var")
				}
			}
		}
	}
}

// Parse func
func Parse(stack types.Stack, item map[string]interface{}) types.RunItem {
	app.App.Mutex.CurrentWorkDirMutex.Lock()
	defer app.App.Mutex.CurrentWorkDirMutex.Unlock()
	os.Chdir(stack.GetWorkdir())
	jsonnetSnippet, jsonnetFiles := func() (string, []string) {
		var jsonnetFiles []string
		switch item["jsonnet"].(type) {
		case string:
			return item["jsonnet"].(string), nil
		case []interface{}:
			var resultJsonnet string
			for _, v := range item["jsonnet"].([]interface{}) {
				jsonnetFiles = append(jsonnetFiles, v.(string))
			}
			return resultJsonnet, jsonnetFiles
		}
		err := fmt.Errorf("Unable to parse run item")
		misc.CheckIfErr(err)
		return "", nil
	}()

	output := new(jsonnetItem)
	output.Jsonnet = jsonnetSnippet
	output.Jsonnet = jsonnetSnippet
	output.Paths = jsonnetFiles
	output.Vars = item["vars"]
	output.Output = item["output"].([]interface{})
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

	return output
}

func processJsonnet(parentWG *sync.WaitGroup, rootObject interface{}, str string) string {
	if parentWG != nil {
		defer parentWG.Done()
	}

	vm := jsonnet.MakeVM()
	vm.TLACode("stack", misc.ToJSON(rootObject))
	result, err := vm.EvaluateSnippet("jsonnet", str)
	misc.CheckIfErr(err)
	return result
}
