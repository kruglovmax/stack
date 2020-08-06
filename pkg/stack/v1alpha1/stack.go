package v1alpha1

import (
	"github.com/davecgh/go-spew/spew"
	config "github.com/kruglovmax/stack/pkg/appconfig"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
)

// Stack type
type Stack struct {
	parentStack *Stack
	config      StackConfig
	appConfig   *config.AppConfig
	runnable    bool

	API       string      `json:"api,omitempty"`
	Name      string      `json:"name,omitempty"`
	Workspace string      `json:"workspace,omitempty"`
	Message   string      `json:"message,omitempty"`
	Tags      []string    `json:"tags,omitempty"`
	Vars      interface{} `json:"vars,omitempty"`
	Libs      []string    `json:"libs,omitempty"`
	Run       Run         `json:"run,omitempty"`
	Stacks    []Stack     `json:"stacks,omitempty"`
	Logs      log.Type    `json:"logs,omitempty"`
}

// Execute func
func (stack *Stack) Execute() {
	if misc.TagsMatcher(stack.Tags, *stack.appConfig.TagPatterns) {
		if stack.runnable {
			log.Logger.Debug().
				Msgf("[RUN] Stack %s", stack.Name)
			for _, v := range stack.Run {
				v.execute(stack)
			}

			log.Logger.Trace().
				Msg(spew.Sdump(stack.GetRealVars()))

			stack.parseStacks()
			for _, v := range stack.Stacks {
				v.Execute()
			}
		}
	}
}

// SetAppConfig func
// func (stack *Stack) SetAppConfig(appConfig *appconfig.AppConfig) {
// 	stack.appConfig = appConfig
// }

// GetConfig func
func (stack Stack) GetConfig() StackConfig {
	return stack.config
}

// GetRealVars func
func (stack Stack) GetRealVars() interface{} {
	return misc.GetRealVars(stack.ToMap())
}

// ToYAML func
func (stack Stack) ToYAML() string {
	return misc.ToYAML(stack)
}

// ToMap func
func (stack Stack) ToMap() (out map[string]interface{}) {
	misc.LoadYAML(misc.ToYAML(stack), &out)
	return out
}

// GetRunItemVars func
func GetRunItemVars(stack Stack, vars interface{}) (result interface{}) {
	switch (vars).(type) {
	case string:
		result = misc.GetObject(stack.GetRealVars(), (vars).(string))
	case map[string]interface{}:
		result = vars
	default:
		result = stack.GetRealVars()
	}
	return result
}

// NewStackFromConfig func
func (stack *Stack) NewStackFromConfig(appConfig *config.AppConfig, config StackConfig, parent *Stack) {
	stack.config = config
	stack.parentStack = parent
	stack.appConfig = appConfig

	stack.Init()
}
