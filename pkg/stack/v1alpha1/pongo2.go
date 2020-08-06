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

// pongo2Item type
type pongo2Item struct {
	item           *interface{}
	stack          *Stack
	templateString string

	Template interface{}   `json:"pongo2,omitempty"`
	Vars     interface{}   `json:"vars,omitempty"`
	Output   []interface{} `json:"output,omitempty"`
}

func (item pongo2Item) RootMap(stack Stack) interface{} {
	vars := GetRunItemVars(stack, item.Vars)
	if vars != nil {
		return misc.GetRealVars(vars)
	}
	return nil
}

// Execute func
func (item pongo2Item) execute(stack *Stack) {
	// html.UnescapeString()
	if err := os.Chdir(stack.Workspace); err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(stack.Workspace))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(err.Error())
	}
	output := bufio.NewScanner(strings.NewReader(processPongo2String(item.RootMap(*stack), item.templateString)))
	stdoutReadDone := make(chan struct{})
	go GetRunItemOutput(stack, item, output, stdoutReadDone, false)
	<-stdoutReadDone
	os.Chdir(*stack.appConfig.Workspace)
}

func (item pongo2Item) getOutput() []interface{} {
	return item.Output
}

func parsePongo2Item(stack *Stack, item *interface{}) pongo2Item {
	outputType := misc.GetRunItemOutputType(*item)
	tmplItem := (*item).(map[string]interface{})
	templateLoader := func() string {
		switch tmplItem["pongo2"].(type) {
		case string:
			return readFileFromPath(stack.GetConfig(), processStringPath(*stack, tmplItem["pongo2"].(string)))
		case []interface{}:
			var resultTemplate string
			for _, path := range tmplItem["pongo2"].([]interface{}) {
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
			Msg("Unable to parse pongo2 item")
		return ""
	}()

	templateProcessor := func() interface{} {
		switch tmplItem["pongo2"].(type) {
		case string:
			if err := os.Chdir(stack.Workspace); err != nil {
				log.Logger.Trace().
					Msg(spew.Sdump(stack.Workspace))
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Fatal().
					Msg(err.Error())
			}
			result := processPongo2String(stack.GetRealVars(), tmplItem["pongo2"].(string))
			os.Chdir(*stack.appConfig.Workspace)
			return result
		case []interface{}:
			var tmpl []interface{}
			for _, v := range tmplItem["pongo2"].([]interface{}) {
				if err := os.Chdir(stack.Workspace); err != nil {
					log.Logger.Trace().
						Msg(spew.Sdump(stack.Workspace))
					log.Logger.Debug().
						Msg(string(debug.Stack()))
					log.Logger.Fatal().
						Msg(err.Error())
				}
				result := processPongo2String(stack.GetRealVars(), v.(string))
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
			Msg("Unable to parse pongo2 item")
		return nil
	}()

	return pongo2Item{
		templateString: templateLoader,
		stack:          stack,
		item:           item,
		Template:       templateProcessor,
		Vars:           tmplItem["vars"],
		Output:         outputType,
	}
}
