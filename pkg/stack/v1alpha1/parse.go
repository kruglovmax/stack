package v1alpha1

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"text/template"

	"github.com/a8m/envsubst"
	"github.com/davecgh/go-spew/spew"
	"github.com/flytam/filenamify"
	gomplate "github.com/hairyhenderson/gomplate/v3"
	gomplateData "github.com/hairyhenderson/gomplate/v3/data"
	gomplateTmpl "github.com/hairyhenderson/gomplate/v3/tmpl"
	"github.com/joeycumines/go-dotnotation/dotnotation"
	pongo2 "gopkg.in/flosch/pongo2.v3"

	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
)

func (stack *Stack) parseRun() {
	input := stack.GetConfig().Run

	switch input.(type) {
	case []interface{}:
		for _, item := range input.([]interface{}) {
			execItem := parseExecItem(stack, &item)
			if execItem != nil {
				stack.Run = append(stack.Run, execItem)
			}
		}
	default:
		stack.Run = nil
	}
}

func (stack *Stack) parseLibs() {
	input := stack.GetConfig().Libs
	for _, item := range input {
		stack.Libs = append(stack.Libs, parseLibsItems(stack, item))
	}
	// stack.Libs = misc.UniqueStrings(stack.Libs)
}

func (stack *Stack) parseStacks() {
	input := stack.GetConfig().Stacks
	for _, item := range input {
		stack.Stacks = append(stack.Stacks, parseStackItems(stack, item, "")...)
	}
}

func parseExecItem(stack *Stack, item *interface{}) runItem {
	switch (*item).(type) {
	case map[string]interface{}:
		switch {
		case (*item).(map[string]interface{})["gomplate"] != nil:
			return parseGomplateItem(stack, item)
		case (*item).(map[string]interface{})["pongo2"] != nil:
			return parsePongo2Item(stack, item)
		case (*item).(map[string]interface{})["script"] != nil:
			return parseScriptItem(stack, item)
		case (*item).(map[string]interface{})["chart"] != nil:
			return parseChartItem(stack, item)
		default:
			return nil
		}
	default:
		return nil
	}
}

func parseLibsItems(stack *Stack, item interface{}) string {
	switch item.(type) {
	case string:
		return processStringPath(*stack, item.(string))
	case map[string]interface{}:
		libItem := item.(map[string]interface{})
		switch {
		case libItem["git"] != nil:
			output, err := filenamify.Filenamify(libItem["git"].(string), filenamify.Options{})
			if err != nil {
				log.Logger.Fatal().
					Msg(err.Error() + "\n" + string(debug.Stack()))
			}
			dirPrefix := filepath.Join(".gitlibs", output)
			dir := filepath.Join(dirPrefix, libItem["commit"].(string))
			cmd := exec.Command("sh", "-c", fmt.Sprintf(`
				mkdir %s -p
				git clone %s %s
				cd %s
				git checkout %s
			`,
				dir,
				libItem["git"].(string), dir,
				dir,
				libItem["commit"].(string),
			))
			cmd.Env = append(os.Environ())
			cmd.Start()
			cmd.Wait()
			return processStringPath(*stack, filepath.Join(dir, libItem["path"].(string)))
		default:
			return ""
		}
	default:
		return ""
	}
}

func isStack(item interface{}) bool {
	switch item.(type) {
	case map[string]interface{}:
		sp := item.(map[string]interface{})
		if sp["name"] != nil && (sp["run"] != nil || sp["stacks"] != nil) {
			return true
		}
	default:
		return false
	}
	return false
}

func parseStackItems(stack *Stack, item interface{}, namePrefix string) (result []Stack) {
	switch item.(type) {
	case string:
		var stackDirs []string
		for _, libDir := range stack.Libs {
			if err := os.Chdir(stack.Workspace); err != nil {
				log.Logger.Trace().
					Msg(spew.Sdump(stack.Workspace))
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Fatal().
					Msg(err.Error())
			}
			matchedDirs := misc.GetDirsByRegexp(filepath.Join(libDir, namePrefix), processString(stack.GetRealVars(), item.(string)))
			os.Chdir(*stack.appConfig.Workspace)
			if matchedDirs != nil {
				for _, dir := range matchedDirs {
					stackDirs = append(stackDirs, filepath.Join(libDir, namePrefix, dir))
				}
				break
			}
		}
		// spew.Dump(stackDirs)
		for _, stackDir := range stackDirs {
			var newStack Stack
			newStack.parentStack = stack
			newStack.FromFile(stack.appConfig, filepath.Join(stackDir, "stack.yaml"), stack)
			result = append(result, newStack)
		}
		return
	case []interface{}:
		for _, v := range item.([]interface{}) {
			switch v.(type) {
			case string:
				result = append(result, parseStackItems(stack, v, namePrefix)...)
			default:
				result = append(result, parseStackItems(stack, v, namePrefix)...)
			}
		}
		return
	case map[string]interface{}:
		if isStack(item) {
			newStackConfig := item.(map[string]interface{})
			if newStackConfig["api"] == nil {
				newStackConfig["api"] = stack.API
			}
			newStackConfig["name"] = namePrefix + newStackConfig["name"].(string)
			var newStack Stack
			newStack.NewStackFromConfig(stack.appConfig, NewConfigFromInterface(newStackConfig), stack)
			result = append(result, newStack)
		} else {
			for k, v := range item.(map[string]interface{}) {
				result = append(result, parseStackItems(stack, v, filepath.Join(namePrefix, k))...)
			}
		}
		return
	default:
		return nil
	}
}

// parseConditions func
func (stack *Stack) parseConditions() {
	input := stack.GetConfig().Conditions
	if input == nil {
		stack.runnable = true
		log.Logger.Debug().
			Bool("runnable", stack.runnable).
			Msgf("Stack %s", stack.Name)
		return
	}
	for _, condition := range input {
		kv := strings.SplitN(condition, "=", 2)
		key := kv[0]
		value := ""
		if len(kv) == 2 {
			value = kv[1]
		}

		actualValue, err := dotnotation.Get(stack.GetRealVars(), "vars."+key)
		if err != nil {
			log.Logger.Debug().
				Msg("parseConditions: " + err.Error())
			stack.runnable = false
			log.Logger.Debug().
				Bool("runnable", stack.runnable).
				Msgf("Stack %s", stack.Name)
			return
		}
		if value != "" && actualValue != value {
			stack.runnable = false
			log.Logger.Debug().
				Bool("runnable", stack.runnable).
				Msgf("Stack %s", stack.Name)
			return
		}

	}
	stack.runnable = true
	log.Logger.Debug().
		Bool("runnable", stack.runnable).
		Msgf("Stack %s", stack.Name)
	return
}

// processString func
func processString(rootMap interface{}, str string) string {
	var gtpl *gomplateTmpl.Template
	root := template.New("root")
	funcMap := gomplate.Funcs(&gomplateData.Data{})

	gtpl = gomplateTmpl.New(root, rootMap)
	funcMap["tpl"] = gtpl.Inline
	funcMap["tmpl"] = func() *gomplateTmpl.Template {
		return gtpl
	}
	root.Funcs(funcMap)
	gtplOut, err := gtpl.Inline(str)
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(rootMap, str))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().Str("Template", str).
			Msg(err.Error())
	}
	return gtplOut
}

// processPongo2String func
func processPongo2String(rootMap interface{}, str string) string {
	// Compile the template first (i. e. creating the AST)
	tpl, err := pongo2.FromString(str)
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(str))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().Str("Template", str).
			Msg(err.Error())
	}

	out, err := tpl.Execute(pongo2.Context{"root": rootMap})
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(str))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().Str("Template", str).
			Msg(err.Error())
	}

	return out
}

func findPathInLibs(libs []string, path string) (fullpath string) {
	for _, libpath := range libs {
		fullpath = filepath.Join(libpath, path)
		if pathIsExists(fullpath) {
			return fullpath
		}
	}
	fullpath = ""
	return
}

// processStringPath func
func processStringPath(stack Stack, str string) string {
	var err error

	str = processString(stack.GetRealVars(), str)

	// environments substitution
	str, err = envsubst.String(str)
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(str))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().Str("Template", str).
			Msg(err.Error())
	}

	if len(strings.TrimRight(str, " ")) == 0 {
		log.Logger.Trace().
			Msg(spew.Sdump(str))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().Str("path", str).
			Msg("Path not found")
	}

	if filepath.IsAbs(str) && pathIsExists(str) {
		return filepath.Clean(str)
	}

	fullpath := findPathInLibs(stack.Libs, str)
	if fullpath == "" {
		log.Logger.Trace().
			Msg(spew.Sdump(str))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().Str("path", str).
			Msg("Path not found")
	}
	return filepath.Clean(fullpath)
}

// pathIsExists returns whether the given file or directory exists
func pathIsExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}

func pathIsDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(path))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().Str("path", path).
			Msg("Path not exists")
		return false
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		return true
	}
	return false
}
