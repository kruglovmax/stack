package v1alpha1

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
	"sigs.k8s.io/yaml"
)

var (
	logMap = map[string]log.Type{
		"json":   log.JSONType,
		"pretty": log.PrettyType,
	}
)

// StackConfig type
type StackConfig struct {
	fileName string

	API        string              `json:"api,omitempty"`
	Name       string              `json:"name,omitempty"`
	Workspace  string              `json:"workspace,omitempty"`
	Message    string              `json:"message,omitempty"`
	Tags       []string            `json:"tags,omitempty"`
	Vars       interface{}         `json:"vars,omitempty"`
	VarsFrom   []map[string]string `json:"varsFrom,omitempty"`
	Libs       []interface{}       `json:"libs,omitempty"`
	Run        interface{}         `json:"run,omitempty"`
	Stacks     []interface{}       `json:"stacks,omitempty"`
	Logs       string              `json:"logs,omitempty"`
	Conditions []string            `json:"conditions,omitempty"`
}

// ToMap func
func (config StackConfig) ToMap() (out map[string]interface{}) {
	misc.LoadYAML(misc.ToYAML(config), &out)
	return out
}

// Init stack
func (stack *Stack) Init() {
	var varsArray []interface{}

	config := stack.GetConfig()

	stackConfigFile := config.fileName
	if stackConfigFile != "" {
		stack.Libs = append(stack.Libs, filepath.Dir(stackConfigFile))
	}
	stack.Libs = append(stack.Libs, *stack.appConfig.Workspace)

	stack.API = config.API

	if stackConfigFile != "" {
		stackdir := filepath.Dir(stackConfigFile)
		ss := strings.TrimLeft(strings.TrimPrefix(stackdir, *stack.appConfig.Workspace), "/")
		namestack := ""
		if ss == "" {
			namestack = filepath.Base(filepath.Dir(stackConfigFile))
		} else {
			namestack = ss
		}
		if config.Name != "" {
			if !strings.Contains(namestack, config.Name) {
				log.Logger.Error().
					Str("name", config.Name).
					Str("namestack", namestack).
					Msg("name field is not valid\n" + string(debug.Stack()))
			}
		}
		stack.Name = namestack
	} else {
		stack.Name = config.Name
	}

	if config.Workspace == "" {
		config.Workspace = "."
	}

	stack.Workspace = processStringPath(*stack, config.Workspace)

	stack.Vars = stack.GetConfig().Vars
	for _, v := range stack.GetConfig().VarsFrom {
		if file, ok := v["file"]; ok {
			var vars interface{}
			misc.LoadYAMLFromFile(processStringPath(*stack, file), &vars)
			varsArray = append(varsArray, vars)
		} else if file, ok := v["sops"]; ok {
			var vars interface{}
			misc.LoadYAMLFromSopsFile(processStringPath(*stack, file), &vars)
			varsArray = append(varsArray, vars)
		}
	}

	for _, vars := range varsArray {
		stack.AddVarsLeft(vars)
	}
	stack.combineStack()

	stack.parseConditions()

	if stack.runnable {

		os.Chdir(stack.Workspace)
		stack.Message = processString(stack.GetRealVars(), config.Message)
		fmt.Fprintf(os.Stderr, stack.Message)

		for _, v := range config.Tags {
			stack.Tags = append(stack.Tags, processString(stack.GetRealVars(), v))
		}
		os.Chdir(*stack.appConfig.Workspace)
		stack.parseLibs()
		stack.Logs = logMap[config.Logs]
		stack.parseRun()
	}
}

// AddVarsLeft func
func (stack *Stack) AddVarsLeft(vars interface{}) {
	stack.Vars = misc.CombineVars(vars, stack.Vars)
}

// AddVarsRight func
func (stack *Stack) AddVarsRight(vars interface{}) {
	stack.Vars = misc.CombineVars(stack.Vars, vars)
}

// ToYAML func
func (config StackConfig) ToYAML() string {
	y, err := yaml.Marshal(config)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	return string(y)
}

// readFileFromPath func
func readFileFromPath(config StackConfig, path string) string {
	var fullPath string
	var result string

	loadTemplateFromWalkPath := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if pathIsExists(path) && !info.IsDir() {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			result = result + string(content) + "\n"
		}
		return nil
	}

	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Join(config.Workspace, path)
	}
	if pathIsExists(fullPath) {
		filepath.Walk(fullPath, loadTemplateFromWalkPath)
		return result
	}

	log.Logger.Trace().
		Msg(spew.Sdump(config))
	log.Logger.Debug().
		Msg(string(debug.Stack()))
	log.Logger.Fatal().Str("path", path).
		Msg("Path not exists")
	return ""
}
