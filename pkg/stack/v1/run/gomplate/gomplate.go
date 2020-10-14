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
	"github.com/kruglovmax/stack/pkg/conditions"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
	"github.com/kruglovmax/stack/pkg/types"
	"sigs.k8s.io/yaml"
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

	rawItem map[string]interface{}
	stack   types.Stack
}

// New func
func New(stack types.Stack, rawItem map[string]interface{}) types.RunItem {
	item := new(gomplateItem)
	item.rawItem = rawItem
	item.stack = stack

	return item
}

// Exec func
func (item *gomplateItem) Exec(parentWG *sync.WaitGroup, stack types.Stack) {
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

	app.App.Mutex.CurrentWorkDirMutex.Lock()
	os.Chdir(stack.GetWorkdir())
	os.Setenv("AWS_TIMEOUT", fmt.Sprint(int64(item.RunTimeout/time.Millisecond)))
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
		stackMap := stack.GetView().(map[string]interface{})
		stackMap["stack"] = stackMap
		vars, err := dotnotation.Get(stackMap, item.Vars.(string))
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
				Msg("Gomplate waiting failed")
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
				case strings.HasPrefix(yml2var, "vars") || strings.HasPrefix(yml2var, "stack.vars"):
					key := strings.TrimPrefix(yml2var, "stack.")
					key = strings.TrimPrefix(strings.TrimPrefix(yml2var, "vars"), ".")
					setVar := gabs.New()
					if key == "" {
						setVar.Set(value)
					} else {
						setVar.SetP(value, key)
					}
					stack.AddRawVarsRight(setVar.Data().(map[string]interface{}))
				case strings.HasPrefix(yml2var, "flags") || strings.HasPrefix(yml2var, "stack.flags"):
					key := strings.TrimPrefix(yml2var, "stack.")
					key = strings.TrimPrefix(strings.TrimPrefix(yml2var, "flags"), ".")
					setVar := gabs.New()
					if key == "" {
						setVar.Set(value)
					} else {
						setVar.SetP(value, key)
					}
					stack.GetFlags().Mux.Lock()
					err := mergo.Merge(&stack.GetFlags().Vars, setVar.Data().(map[string]interface{}), mergo.WithOverwriteWithEmptyValue)
					misc.CheckIfErr(err)
					stack.GetFlags().Mux.Unlock()
				case strings.HasPrefix(yml2var, "locals") || strings.HasPrefix(yml2var, "stack.locals"):
					key := strings.TrimPrefix(yml2var, "stack.")
					key = strings.TrimPrefix(strings.TrimPrefix(yml2var, "locals"), ".")
					setVar := gabs.New()
					if key == "" {
						setVar.Set(value)
					} else {
						setVar.SetP(value, key)
					}
					stack.GetLocals().Mux.Lock()
					err := mergo.Merge(&stack.GetLocals().Vars, setVar.Data().(map[string]interface{}), mergo.WithOverwriteWithEmptyValue)
					misc.CheckIfErr(err)
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
				case strings.HasPrefix(str2var, "vars.") || strings.HasPrefix(str2var, "stack.vars."):
					key := strings.TrimPrefix(str2var, "stack.")
					key = strings.TrimPrefix(str2var, "vars.")
					setVar := gabs.New()
					setVar.SetP(parsedString, key)
					stack.AddRawVarsRight(setVar.Data().(map[string]interface{}))
				case strings.HasPrefix(str2var, "flags.") || strings.HasPrefix(str2var, "stack.flags."):
					key := strings.TrimPrefix(str2var, "stack.")
					key = strings.TrimPrefix(str2var, "flags.")
					setVar := gabs.New()
					setVar.SetP(parsedString, key)
					stack.GetFlags().Mux.Lock()
					err := mergo.Merge(&stack.GetFlags().Vars, setVar.Data().(map[string]interface{}), mergo.WithOverwriteWithEmptyValue)
					misc.CheckIfErr(err)
					stack.GetFlags().Mux.Unlock()
				case strings.HasPrefix(str2var, "locals.") || strings.HasPrefix(str2var, "stack.locals."):
					key := strings.TrimPrefix(str2var, "stack.")
					key = strings.TrimPrefix(str2var, "locals.")
					setVar := gabs.New()
					setVar.SetP(parsedString, key)
					stack.GetLocals().Mux.Lock()
					err := mergo.Merge(&stack.GetLocals().Vars, setVar.Data().(map[string]interface{}), mergo.WithOverwriteWithEmptyValue)
					misc.CheckIfErr(err)
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

func (item *gomplateItem) parse() {
	app.App.Mutex.CurrentWorkDirMutex.Lock()
	defer app.App.Mutex.CurrentWorkDirMutex.Unlock()
	os.Chdir(item.stack.GetWorkdir())
	templateLoader := func() string {
		switch item.rawItem["gomplate"].(type) {
		case string:
			return item.rawItem["gomplate"].(string)
		case []interface{}:
			var resultTemplate string
			for _, path := range item.rawItem["gomplate"].([]interface{}) {
				path := path.(string)
				stackMap := item.stack.GetView().(map[string]interface{})
				stackMap["stack"] = stackMap
				computed, err := cel.ComputeCEL(path, stackMap)
				if _, ok := computed.(string); err == nil && ok {
					path = computed.(string)
				}
				if !filepath.IsAbs(path) {
					path = filepath.Join(item.stack.GetWorkdir(), path)
				}
				resultTemplate = resultTemplate + misc.ReadFileFromPath(path)
			}
			return resultTemplate
		}
		err := fmt.Errorf("Unable to parse run item")
		misc.CheckIfErr(err)
		return ""
	}()

	item.Template = templateLoader
	item.Vars = item.rawItem["vars"]
	item.Output = item.rawItem["output"].([]interface{})
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
		misc.CheckIfErr(err)
	}
	waitTimeout := item.rawItem["waitTimeout"]
	item.WaitTimeout = *app.App.Config.DefaultTimeout
	if waitTimeout != nil {
		item.WaitTimeout, err = time.ParseDuration(waitTimeout.(string))
		misc.CheckIfErr(err)
	}
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
