package v1alpha1

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime/debug"

	"github.com/davecgh/go-spew/spew"
	jsonschema "github.com/xeipuuv/gojsonschema"

	config "github.com/kruglovmax/stack/pkg/appconfig"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
)

// NewConfigFromInterface func
func NewConfigFromInterface(configMap interface{}) (config StackConfig) {

	validation, err := ConfigSchema.Validate(jsonschema.NewGoLoader(configMap))
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(configMap))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(fmt.Sprintf("cannot validate: %s\n", err))
	}
	if !validation.Valid() {
		errs := ""
		for _, e := range validation.Errors() {
			errs = errs + "\n" + e.String()
		}
		log.Logger.Trace().
			Msg(spew.Sdump(configMap))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(fmt.Sprintf("stack file is not valid: %s\n", errs))
	}

	misc.LoadYAML(misc.ToYAML(configMap), &config)
	return config
}

// NewConfigFromBytes func
func NewConfigFromBytes(yamlConfig []byte) (config StackConfig) {
	var preConfig interface{}
	misc.LoadYAML(string(yamlConfig), &preConfig)
	config = NewConfigFromInterface(preConfig)

	return config
}

/*
NewConfigFromFile StackConfig
*/
func NewConfigFromFile(fileName string) (config StackConfig) {
	fileName, err := filepath.Abs(fileName)
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(fileName))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(err.Error())
	}
	log.Logger.Info().Str("file", fileName).Msg("Reading stack from")

	content, ioErr := ioutil.ReadFile(fileName)
	if ioErr != nil {
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Str("fileName", fileName).
			Msg(ioErr.Error())
	}

	config = NewConfigFromBytes(content)

	config.fileName = fileName
	if config.Name == "" {
		config.Name = filepath.Base(filepath.Dir(fileName))
	}

	if config.Workspace == "" {
		config.Workspace = "."
	}

	return config
}

/*
NewStackFromFile Stack
*/
func NewStackFromFile(appConfig *config.AppConfig, fileName string) (stack Stack) {
	stack.NewStackFromConfig(appConfig, NewConfigFromFile(fileName), nil)
	return
	// return NewStackFromConfig(NewConfigFromFile(fileName), nil)
}

/*
FromFile Stack
*/
func (stack *Stack) FromFile(appConfig *config.AppConfig, fileName string, parent *Stack) {
	stack.NewStackFromConfig(appConfig, NewConfigFromFile(fileName), parent)
}
