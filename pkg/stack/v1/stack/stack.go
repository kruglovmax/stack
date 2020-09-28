package stack

import (
	"fmt"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"github.com/imdario/mergo"
	"github.com/kruglovmax/stack/pkg/app"
	"github.com/kruglovmax/stack/pkg/conditions"
	"github.com/kruglovmax/stack/pkg/consts"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
	"github.com/kruglovmax/stack/pkg/stack/v1/libs"
	"github.com/kruglovmax/stack/pkg/stack/v1/run/parser"
	"github.com/kruglovmax/stack/pkg/stack/v1/schema"
	"github.com/kruglovmax/stack/pkg/stack/v1/vars"
	"github.com/kruglovmax/stack/pkg/types"
	jsonschema "github.com/xeipuuv/gojsonschema"
	"k8s.io/helm/pkg/strvals"
)

// Stack type
type Stack struct {
	// WaitGroup
	stacskWG   sync.WaitGroup
	preExecWG  sync.WaitGroup
	execWG     sync.WaitGroup
	postExecWG sync.WaitGroup

	// Mutex
	getViewMutex sync.Mutex

	// config
	config        stackInputYAML
	runItemParser types.RunItemParser
	parentStack   types.Stack
	stackID       string

	// API
	API         string
	Name        string
	Vars        *types.StackVars
	Flags       *types.StackFlags
	Locals      *types.StackLocals
	Workdir     string
	Libs        []string
	PreRun      []types.RunItem
	Run         []types.RunItem
	PostRun     []types.RunItem
	Stacks      []types.Stack
	Status      *types.StacksStatus
	When        string
	Wait        string
	WaitTimeout time.Duration
	WaitGroups  []*sync.WaitGroup
}

// YAML view of StackConfig
type stackInputYAML struct {
	workdir string

	API         string                 `json:"api,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Vars        map[string]interface{} `json:"vars,omitempty"`
	VarsFrom    []map[string]string    `json:"varsFrom,omitempty"`
	Flags       map[string]interface{} `json:"flags,omitempty"`
	Locals      map[string]interface{} `json:"locals,omitempty"`
	Libs        []interface{}          `json:"libs,omitempty"`
	PreRun      []interface{}          `json:"preRun,omitempty"`
	Run         []interface{}          `json:"run,omitempty"`
	PostRun     []interface{}          `json:"postRun,omitempty"`
	Stacks      []interface{}          `json:"stacks,omitempty"`
	When        string                 `json:"when,omitempty"`
	Wait        string                 `json:"wait,omitempty"`
	WaitTimeout string                 `json:"waitTimeout,omitempty"`
	WaitGroups  []string               `json:"waitGroups,omitempty"`
}

type stackOutputValues struct {
	API    string                 `json:"api,omitempty"`
	ID     string                 `json:"id,omitempty"`
	Name   string                 `json:"name,omitempty"`
	Vars   map[string]interface{} `json:"vars,omitempty"`
	Flags  map[string]interface{} `json:"flags,omitempty"`
	Locals map[string]interface{} `json:"locals,omitempty"`
	Status map[string]string      `json:"status,omitempty"`
}

// AddRawVarsLeft func
func (stack *Stack) AddRawVarsLeft(v map[string]interface{}) {
	stack.Vars.Mux.Lock()
	defer stack.Vars.Mux.Unlock()
	stack.Vars = vars.CombineVars(vars.ParseVars(v), stack.Vars)
}

// AddRawVarsRight func
func (stack *Stack) AddRawVarsRight(v map[string]interface{}) {
	stack.Vars.Mux.Lock()
	defer stack.Vars.Mux.Unlock()
	stack.Vars = vars.CombineVars(stack.Vars, vars.ParseVars(v))
}

// GetAPI func
func (stack *Stack) GetAPI() string {
	return stack.API
}

// GetLibs func
func (stack *Stack) GetLibs() []string {
	return stack.Libs
}

// GetName func
func (stack *Stack) GetName() string {
	return stack.Name
}

// GetRunItemsParser func
func (stack *Stack) GetRunItemsParser() types.RunItemParser {
	return stack.runItemParser
}

// GetStackID func
func (stack *Stack) GetStackID() string {
	return stack.stackID
}

// GetVars func
func (stack *Stack) GetVars() *types.StackVars {
	return stack.Vars
}

// GetFlags func
func (stack *Stack) GetFlags() *types.StackFlags {
	return stack.Flags
}

// GetLocals func
func (stack *Stack) GetLocals() *types.StackLocals {
	return stack.Locals
}

// GetView func
func (stack *Stack) GetView() (result interface{}) {
	var output stackOutputValues

	stack.getViewMutex.Lock()
	stack.Vars.Mux.Lock()
	stack.Flags.Mux.Lock()
	stack.Locals.Mux.Lock()
	stack.Status.Mux.Lock()
	defer stack.getViewMutex.Unlock()
	defer stack.Vars.Mux.Unlock()
	defer stack.Flags.Mux.Unlock()
	defer stack.Locals.Mux.Unlock()
	defer stack.Status.Mux.Unlock()

	output.API = stack.API
	output.Name = stack.Name
	output.Vars = stack.Vars.Vars
	output.Flags = stack.Flags.Vars
	output.Locals = stack.Locals.Vars
	output.Status = stack.Status.StacksStatus
	output.ID = stack.GetStackID()

	result = misc.ToInterface(output)

	return
}

// GetWaitTimeout func
func (stack *Stack) GetWaitTimeout() time.Duration {
	return stack.WaitTimeout
}

// GetWorkdir func
func (stack *Stack) GetWorkdir() string {
	return stack.Workdir
}

// ParseStacks func
func ParseStacks(stack types.Stack, input []interface{}) (output []types.Stack) {
	for _, item := range input {
		output = append(output, parseStackItems(stack, item, "")...)
	}
	return
}

// LoadFromString reads stack from yaml or json to self struct
func (stack *Stack) LoadFromString(stackYAML string, parentStack types.Stack) {
	log.Logger.Info().Str("inline", "YAML").Msg(consts.MessagesReadingStackFrom)

	// schema validation
	var tmpStructForValidation interface{}
	misc.LoadYAML(stackYAML, &tmpStructForValidation)
	validation, err := schema.ConfigSchema.Validate(jsonschema.NewGoLoader(tmpStructForValidation))
	misc.CheckIfErr(err)
	if !validation.Valid() {
		var errs string
		for _, e := range validation.Errors() {
			errs = errs + "\n" + e.String()
		}
		err := fmt.Errorf(consts.MessageBadStackErr, errs)
		misc.CheckIfErr(err)
	}

	misc.LoadYAML(stackYAML, &stack.config)
	switch stack.config.API {
	case "v1":
		stack.runItemParser = parser.RunItemParser
		stack.parentStack = parentStack
		stack.Name = stack.config.Name
		stack.Workdir = parentStack.GetWorkdir()
		parseInputYAML(stack, stack.config, parentStack)
	default:
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Str("YAML", "\n"+stackYAML).
			Msg(consts.MessageBadStackUnsupportedAPI)
	}
	stack.Status.Mux.Lock()
	stack.Status.StacksStatus[stack.stackID] = "Loaded"
	stack.Status.Mux.Unlock()
	return
}

// LoadFromFile reads stack from yaml or json to self struct
func (stack *Stack) LoadFromFile(stackFile string, parentStack types.Stack) {
	log.Logger.Info().Str("file", stackFile).Msg(consts.MessagesReadingStackFrom)

	// schema validation
	var tmpStructForValidation interface{}
	misc.LoadYAMLFromFile(stackFile, &tmpStructForValidation)
	validation, err := schema.ConfigSchema.Validate(jsonschema.NewGoLoader(tmpStructForValidation))
	misc.CheckIfErr(err)
	if !validation.Valid() {
		var errs string
		for _, e := range validation.Errors() {
			errs = errs + "\n" + e.String()
		}
		err := fmt.Errorf(consts.MessageBadStackErr, errs)
		misc.CheckIfErr(err)
	}

	misc.LoadYAMLFromFile(stackFile, &stack.config)
	switch stack.config.API {
	case "v1":
		stack.runItemParser = parser.RunItemParser
		stack.parentStack = parentStack
		stack.Name = misc.GetDirName(stackFile)
		stack.Workdir = misc.GetDirPath(stackFile)
		parseInputYAML(stack, stack.config, parentStack)
	default:
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Str("file", stackFile).
			Msg(consts.MessageBadStackUnsupportedAPI)
	}
	stack.Status.Mux.Lock()
	stack.Status.StacksStatus[stack.stackID] = "Loaded"
	stack.Status.Mux.Unlock()
	return
}

// PreExec func
func (stack *Stack) PreExec(parentWG *sync.WaitGroup) {
	if parentWG != nil {
		defer parentWG.Done()
	}
	if len(stack.PreRun) > 0 {
		log.Logger.Info().Str("Stack", stack.GetWorkdir()).Msg("preRun")
	}
	stack.Status.Mux.Lock()
	stack.Status.StacksStatus[stack.stackID] = "PreRun"
	stack.Status.Mux.Unlock()
	for _, runItem := range stack.PreRun {
		var wg sync.WaitGroup
		wg.Add(1)
		go runItem.Exec(&wg, stack)
		wg.Wait()
	}
}

// Exec func
func (stack *Stack) Exec(parentWG *sync.WaitGroup) {
	if parentWG != nil {
		defer parentWG.Done()
	}
	if len(stack.Run) > 0 {
		log.Logger.Info().Str("Stack", stack.GetWorkdir()).Msg("Run")
	}
	stack.Status.Mux.Lock()
	stack.Status.StacksStatus[stack.stackID] = "Run"
	stack.Status.Mux.Unlock()
	for _, runItem := range stack.Run {
		var wg sync.WaitGroup
		wg.Add(1)
		go runItem.Exec(&wg, stack)
		wg.Wait()
	}
}

// PostExec func
func (stack *Stack) PostExec(parentWG *sync.WaitGroup) {
	if parentWG != nil {
		defer parentWG.Done()
	}
	if len(stack.PostRun) > 0 {
		log.Logger.Info().Str("Stack", stack.GetWorkdir()).Msg("postRun")
	}
	stack.Status.Mux.Lock()
	stack.Status.StacksStatus[stack.stackID] = "PostRun"
	stack.Status.Mux.Unlock()
	for _, runItem := range stack.PostRun {
		var wg sync.WaitGroup
		wg.Add(1)
		go runItem.Exec(&wg, stack)
		wg.Wait()
	}
}

// Start func
func (stack *Stack) Start(parentWG *sync.WaitGroup) {
	if parentWG != nil {
		defer parentWG.Done()
	}

	stack.preExecWG.Add(1)
	go stack.PreExec(&stack.preExecWG)
	stack.preExecWG.Wait()

	if !conditions.Wait(stack, stack.Wait, stack.WaitTimeout) {
		return
	}
	if !conditions.When(stack, stack.When) {
		return
	}

	stack.execWG.Add(1)
	go stack.Exec(&stack.execWG)
	stack.execWG.Wait()

	// Start sub stacks
	stack.Status.Mux.Lock()
	stack.Status.StacksStatus[stack.stackID] = "ParseChildStacks"
	stack.Status.Mux.Unlock()
	stack.Stacks = ParseStacks(stack, stack.config.Stacks)
	stack.Status.Mux.Lock()
	stack.Status.StacksStatus[stack.stackID] = "RunChildStacks"
	stack.Status.Mux.Unlock()
	for _, stackItem := range stack.Stacks {
		stack.stacskWG.Add(1)
		go stackItem.Start(&stack.stacskWG)
	}
	stack.stacskWG.Wait()

	stack.postExecWG.Add(1)
	go stack.PostExec(&stack.postExecWG)
	stack.postExecWG.Wait()

	stack.Status.Mux.Lock()
	stack.Status.StacksStatus[stack.stackID] = "Done"
	stack.Status.Mux.Unlock()
	for _, wg := range stack.WaitGroups {
		wg.Done()
	}
}

func parseInputYAML(stack *Stack, input stackInputYAML, parentStack types.Stack) {
	stack.API = input.API

	switch parentStack {
	case nil:
		stack.Vars = vars.ParseVars(input.Vars)
	default:
		stack.Vars = vars.CombineVars(parentStack.GetVars(), vars.ParseVars(input.Vars))
	}

	varsArray := make([]map[string]interface{}, 0, len(input.VarsFrom)+len(*app.App.Config.VarFiles))
	for _, v := range input.VarsFrom {
		if file, ok := v["file"]; ok {
			var varsMap map[string]interface{}
			misc.LoadYAMLFromFile(filepath.Join(stack.Workdir, file), &varsMap)
			varsArray = append(varsArray, varsMap)
		} else if file, ok := v["sops"]; ok {
			var varsMap map[string]interface{}
			misc.LoadYAMLFromSopsFile(filepath.Join(stack.Workdir, file), &varsMap)
			varsArray = append(varsArray, varsMap)
		}
	}

	if parentStack == nil {
		for _, varsFile := range *app.App.Config.VarFiles {
			var varsMap map[string]interface{}
			misc.LoadYAMLFromFile(filepath.Join(stack.Workdir, varsFile), &varsMap)
			varsArray = append(varsArray, varsMap)
		}
		for _, str := range *app.App.Config.CLIValues {
			varsMap, err := strvals.Parse(str)
			misc.CheckIfErr(err)
			varsArray = append(varsArray, varsMap)
		}
	}

	for _, v := range varsArray {
		stack.AddRawVarsLeft(v)
	}

	stack.Flags = vars.FlagsGlobal
	stack.GetFlags().Mux.Lock()
	mergo.Merge(&stack.Flags.Vars, input.Flags)
	stack.GetFlags().Mux.Unlock()

	stack.Locals = new(types.StackLocals)
	stack.Locals.Vars = input.Locals

	stack.Libs = libs.ParseAndInitLibs(input.Libs, stack.Workdir)
	stack.PreRun = stack.GetRunItemsParser().ParseRun(stack, input.PreRun)
	stack.Run = stack.GetRunItemsParser().ParseRun(stack, input.Run)
	stack.PostRun = stack.GetRunItemsParser().ParseRun(stack, input.PostRun)
	stack.When = input.When
	stack.Wait = input.Wait
	waitTimeout := input.WaitTimeout
	stack.WaitTimeout = *app.App.Config.DefaultTimeout
	if waitTimeout != "" {
		var err error
		stack.WaitTimeout, err = time.ParseDuration(waitTimeout)
		misc.CheckIfErr(err)
	}
	stack.Status = app.App.StacksStatus
	stack.stackID = app.NewStackID()
	for _, wgKey := range input.WaitGroups {
		stack.WaitGroups = append(stack.WaitGroups, conditions.WaitGroupAdd(stack, wgKey))
	}
}

func parseStackItems(stack types.Stack, item interface{}, namePrefix string) (output []types.Stack) {
	switch item.(type) {
	case string:
		var stackDirs []string
		for _, libDir := range stack.GetLibs() {
			matchedDirs := misc.GetDirsByRegexp(filepath.Join(libDir, namePrefix), item.(string))
			if matchedDirs != nil {
				for _, dir := range matchedDirs {
					stackDirs = append(stackDirs, filepath.Join(libDir, namePrefix, dir))
				}
				break
			}
		}
		if len(stackDirs) == 0 {
			log.Logger.Fatal().
				Str("In Stack", stack.GetWorkdir()).
				Interface("SubStacks", item).
				Msg("Not found")
		}
		for _, stackDir := range stackDirs {
			newStack := new(Stack)
			newStack.runItemParser = parser.RunItemParser
			newStack.parentStack = stack
			newStack.LoadFromFile(misc.FindStackFileInDir(stackDir), stack)
			output = append(output, newStack)
		}
		return
	case []interface{}:
		for _, v := range item.([]interface{}) {
			output = append(output, parseStackItems(stack, v, namePrefix)...)
		}
		return
	case map[string]interface{}:
		if isStack(item) { // parse inline Stack
			newStackConfig := item.(map[string]interface{})
			if newStackConfig["api"] == nil {
				newStackConfig["api"] = stack.GetAPI()
			}
			newStack := new(Stack)
			newStack.runItemParser = parser.RunItemParser
			newStack.LoadFromString(misc.ToYAML(newStackConfig), stack)
			output = append(output, newStack)
		} else {
			for k, v := range item.(map[string]interface{}) {
				output = append(output, parseStackItems(stack, v, filepath.Join(namePrefix, k))...)
			}
		}
		return
	}
	return
}

func isStack(item interface{}) bool {
	switch item.(type) {
	case map[string]interface{}:
		sp := item.(map[string]interface{})
		if sp["name"] != nil && (sp["run"] != nil || sp["stacks"] != nil || sp["flags"] != nil) {
			return true
		}
	default:
		return false
	}
	return false
}
