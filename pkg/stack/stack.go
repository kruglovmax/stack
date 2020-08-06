package stack

import (
	"github.com/kruglovmax/stack/pkg/stack/v1alpha1"
)

// RootStack instance
var (
	RootStack *v1alpha1.Stack
)

func init() {
	RootStack = new(v1alpha1.Stack)
}
