package gomplate

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/davecgh/go-spew/spew"
	gomplate "github.com/hairyhenderson/gomplate/v3"
	gomplateData "github.com/hairyhenderson/gomplate/v3/data"
	gomplateTmpl "github.com/hairyhenderson/gomplate/v3/tmpl"
	"github.com/imdario/mergo"
	"github.com/joeycumines/go-dotnotation/dotnotation"
	"github.com/kruglovmax/stack/pkg/app"
	"github.com/kruglovmax/stack/pkg/cel"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
	"github.com/kruglovmax/stack/pkg/types"
	"gopkg.in/yaml.v2"
)

// gomplateItem type
type gomplateItem struct {
	Template    string        `json:"gomplate,omitempty"`
	Vars        interface{}   `json:"vars,omitempty"`
	Output      []interface{} `json:"output,omitempty"`
	When        string        `json:"when,omitempty"`
	Wait        string        `json:"wait,omitempty"`
	RunTimeout  time.Duration `json:"runTimeout,omitempty"`
	WaitTimeout time.Duration `json:"waitTimeout,omitempty"`
}

// Exec func
func (item *gomplateItem) Exec(parentWG *sync.WaitGroup, stack types.Stack) {
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
	var parsedString string
	switch item.Vars.(type) {
	case map[string]interface{}:
		var wg sync.WaitGroup
		wg.Add(1)
		parsedString = processString(&wg, item.Vars.(map[string]interface{}), item.Template)
		if misc.WaitTimeout(&wg, item.RunTimeout) {
			log.Logger.Fatal().
				Str("stack", stack.GetWorkdir()).
				Str("timeout", fmt.Sprint(item.RunTimeout)).
				Msg("Gomplate waiting failed")
		}
	case string:
		vars, err := dotnotation.Get(stack.GetView(), item.Vars.(string))
		misc.CheckIfErr(err)
		var wg sync.WaitGroup
		wg.Add(1)
		parsedString = processString(&wg, vars, item.Template)
		if misc.WaitTimeout(&wg, item.RunTimeout) {
			log.Logger.Fatal().
				Str("stack", stack.GetWorkdir()).
				Str("timeout", fmt.Sprint(item.RunTimeout)).
				Msg("Gomplate waiting failed")
		}
	case nil:
		var wg sync.WaitGroup
		wg.Add(1)
		parsedString = processString(&wg, stack.GetView(), item.Template)
		if misc.WaitTimeout(&wg, item.RunTimeout) {
			log.Logger.Fatal().
				Str("stack", stack.GetWorkdir()).
				Str("timeout", fmt.Sprint(item.RunTimeout)).
				Msg("Script waiting failed")
		}
	default:
		log.Logger.Trace().
			Msg(spew.Sdump(item))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Unable to parse run item. Bad vars key")
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
	templateLoader := func() string {
		switch item["gomplate"].(type) {
		case string:
			return item["gomplate"].(string)
		case []interface{}:
			var resultTemplate string
			for _, path := range item["gomplate"].([]interface{}) {
				path := path.(string)
				if !filepath.IsAbs(path) {
					path = filepath.Join(stack.GetWorkdir(), path)
				}
				resultTemplate = resultTemplate + misc.ReadFileFromPath(path)
			}
			return resultTemplate
		}
		err := fmt.Errorf("Unable to parse run item")
		misc.CheckIfErr(err)
		return ""
	}()

	output := new(gomplateItem)
	output.Template = templateLoader
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

func processString(parentWG *sync.WaitGroup, rootObject interface{}, str string) string {
	if parentWG != nil {
		defer parentWG.Done()
	}

	var gtpl *gomplateTmpl.Template
	root := template.New("root")
	funcMap := gomplate.Funcs(&gomplateData.Data{})

	gtpl = gomplateTmpl.New(root, rootObject)
	funcMap["tpl"] = gtpl.Inline
	funcMap["tmpl"] = func() *gomplateTmpl.Template {
		return gtpl
	}
	root.Funcs(funcMap)
	gtplOut, err := gtpl.Inline(str)
	log.Logger.Trace().
		Str("rootMap", spew.Sprint(rootObject)).
		Msg("")
	misc.CheckIfErr(err)
	return gtplOut
}
