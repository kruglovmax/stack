package pongo2

import (
	"sync"
	"time"

	"github.com/kruglovmax/stack/pkg/types"
)

// pongo2Item type
type pongo2Item struct {
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
	item := new(pongo2Item)
	item.rawItem = rawItem
	item.stack = stack

	return item
}

// Exec func
func (item *pongo2Item) Exec(parentWG *sync.WaitGroup) {
}
