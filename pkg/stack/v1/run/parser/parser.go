package parser

import (
	"github.com/kruglovmax/stack/pkg/stack/v1/run/gitclone"
	"github.com/kruglovmax/stack/pkg/stack/v1/run/gomplate"
	"github.com/kruglovmax/stack/pkg/stack/v1/run/group"
	"github.com/kruglovmax/stack/pkg/stack/v1/run/pongo2"
	"github.com/kruglovmax/stack/pkg/stack/v1/run/script"
	"github.com/kruglovmax/stack/pkg/types"
)

// RunItemParser instance
var RunItemParser *runItemParser

type runItemParser struct {
}

func init() {
	RunItemParser = new(runItemParser)
}

// ParseRun func
func (parser *runItemParser) ParseRun(stack types.Stack, input []interface{}) (output []types.RunItem) {
	output = make([]types.RunItem, 0, len(input))
	for _, item := range input {
		runItem := parser.ParseRunItem(stack, item)
		if runItem != nil {
			output = append(output, runItem)
		}
	}
	return
}

// ParseRunItem func
func (parser *runItemParser) ParseRunItem(stack types.Stack, item interface{}) (output types.RunItem) {
	switch item.(type) {
	case map[string]interface{}:
		switch {
		case item.(map[string]interface{})["gomplate"] != nil:
			output = gomplate.Parse(stack, item.(map[string]interface{}))
		case item.(map[string]interface{})["pongo2"] != nil:
			output = pongo2.Parse(stack, item.(map[string]interface{}))
		case item.(map[string]interface{})["script"] != nil:
			output = script.Parse(stack, item.(map[string]interface{}))
		case item.(map[string]interface{})["gitclone"] != nil:
			output = gitclone.Parse(stack, item.(map[string]interface{}))
		case item.(map[string]interface{})["group"] != nil:
			output = group.Parse(stack, item.(map[string]interface{}))
		}
	}
	return
}
