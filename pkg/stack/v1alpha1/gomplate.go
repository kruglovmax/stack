package v1alpha1

import (
	"bufio"
	"os"
	"runtime/debug"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/misc"
)

// gomplateItem type
type gomplateItem struct {
	item           *interface{}
	stack          *Stack
	templateString string

	Template interface{}   `json:"gomplate,omitempty"`
	Vars     interface{}   `json:"vars,omitempty"`
	Output   []interface{} `json:"output,omitempty"`
}

func (item gomplateItem) RootMap(stack Stack) interface{} {
	vars := GetRunItemVars(stack, item.Vars)
	if vars != nil {
		return misc.GetRealVars(vars)
	}
	return nil
}

// Execute func
func (item gomplateItem) execute(stack *Stack) {
	// html.UnescapeString()
	if err := os.Chdir(stack.Workspace); err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(stack.Workspace))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(err.Error())
	}
	output := bufio.NewScanner(strings.NewReader(processString(item.RootMap(*stack), item.templateString)))
	stdoutReadDone := make(chan struct{})
	go GetRunItemOutput(stack, item, output, stdoutReadDone, false)
	<-stdoutReadDone
	os.Chdir(*stack.appConfig.Workspace)
}

func (item gomplateItem) getOutput() []interface{} {
	return item.Output
}

func parseGomplateItem(stack *Stack, item *interface{}) gomplateItem {
	outputType := misc.GetRunItemOutputType(*item)
	tmplItem := (*item).(map[string]interface{})
	templateLoader := func() string {
		switch tmplItem["gomplate"].(type) {
		case string:
			return tmplItem["gomplate"].(string)
		case []interface{}:
			var resultTemplate string
			for _, path := range tmplItem["gomplate"].([]interface{}) {
				resultTemplate = resultTemplate + readFileFromPath(stack.GetConfig(),
					processStringPath(*stack, path.(string)))
			}
			return resultTemplate
		}
		log.Logger.Trace().
			Msg(spew.Sdump(item))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Unable to parse run item")
		return ""
	}()

	templateProcessor := func() interface{} {
		switch tmplItem["gomplate"].(type) {
		case string:
			if err := os.Chdir(stack.Workspace); err != nil {
				log.Logger.Trace().
					Msg(spew.Sdump(stack.Workspace))
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Fatal().
					Msg(err.Error())
			}
			result := processString(stack.GetRealVars(), tmplItem["gomplate"].(string))
			os.Chdir(*stack.appConfig.Workspace)
			return result
		case []interface{}:
			var tmpl []interface{}
			for _, v := range tmplItem["gomplate"].([]interface{}) {
				if err := os.Chdir(stack.Workspace); err != nil {
					log.Logger.Trace().
						Msg(spew.Sdump(stack.Workspace))
					log.Logger.Debug().
						Msg(string(debug.Stack()))
					log.Logger.Fatal().
						Msg(err.Error())
				}
				result := processString(stack.GetRealVars(), v.(string))
				os.Chdir(*stack.appConfig.Workspace)
				tmpl = append(tmpl, result)
			}
			return tmpl
		}
		log.Logger.Trace().
			Msg(spew.Sdump(item))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("Unable to parse run item")
		return nil
	}()

	return gomplateItem{
		templateString: templateLoader,
		stack:          stack,
		item:           item,
		Template:       templateProcessor,
		Vars:           tmplItem["vars"],
		Output:         outputType,
	}
}
