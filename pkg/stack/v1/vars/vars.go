package vars

import (
	"runtime/debug"

	"github.com/davecgh/go-spew/spew"

	"github.com/kruglovmax/stack/pkg/consts"
	"github.com/kruglovmax/stack/pkg/log"
	"github.com/kruglovmax/stack/pkg/types"
)

// StackVarsModifiers type
type StackVarsModifiers struct {
	Update bool
	Clear  bool
	Weak   bool
}

var (
	// FlagsGlobal var
	FlagsGlobal *types.StackFlags

	// VarsSuffixDelimeter var
	VarsSuffixDelimeter = "^"

	// VarsSuffixes var
	VarsSuffixes = map[string]string{
		"Update": "+",
		"Clear":  "-",
		"Weak":   "~",
	}

	thisVarModifiersSuffix = "_modifiers"
)

func init() {
	if FlagsGlobal == nil {
		FlagsGlobal = new(types.StackFlags)
		if FlagsGlobal.Vars == nil {
			FlagsGlobal.Vars = make(map[string]interface{})
		}
	}
}

// ParseVars func
func ParseVars(varsFromConfig map[string]interface{}) *types.StackVars {
	return parseVars(varsFromConfig, nil)
}

func parseVars(varsFromConfig map[string]interface{}, parentVarModifiers *StackVarsModifiers) (stackVars *types.StackVars) {
	stackVars = new(types.StackVars)
	modifiers := make(map[string]interface{})
	vars := make(map[string]interface{})

	for varRawName, varValue := range varsFromConfig {
		varName, varModifiers := ParseVarModifiers(varRawName, parentVarModifiers)
		switch varValue.(type) {
		case map[string]interface{}:
			stackSubVars := parseVars(varValue.(map[string]interface{}), &varModifiers)
			_, ok := modifiers[varName]
			if ok {
				log.Logger.Trace().
					Msg(spew.Sdump(vars))
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Fatal().
					Str("var", varName).
					Msg(consts.MessageVarsDoubleDefinition)
			}
			modifiers[varName] = stackSubVars.Modifiers
			modifiers[varName+thisVarModifiersSuffix] = varModifiers
			vars[varName] = stackSubVars.Vars
		default:
			_, ok := modifiers[varName]
			if ok {
				log.Logger.Trace().
					Msg(spew.Sdump(vars))
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Fatal().
					Str("var", varName).
					Msg(consts.MessageVarsDoubleDefinition)
			}
			modifiers[varName+thisVarModifiersSuffix] = varModifiers
			vars[varName] = varValue
		}
	}
	stackVars.Vars = vars
	stackVars.Modifiers = modifiers
	return
}

// ParseVarModifiers func
// varRawName - test+
func ParseVarModifiers(varRawName string, parentVarModifiers *StackVarsModifiers) (varName string, modifiers StackVarsModifiers) {
	varName = varRawName
	if parentVarModifiers != nil {
		modifiers.Weak = parentVarModifiers.Weak
	}
Loop:
	for {
		sfx := varName[len(varName)-1:]
		varName = varName[:len(varName)-1]
		switch sfx {
		case VarsSuffixes["Update"]:
			if !(modifiers.Update ||
				modifiers.Weak) &&
				(parentVarModifiers == nil || (parentVarModifiers != nil && parentVarModifiers.Update)) {
				if modifiers.Clear {
					log.Logger.Debug().
						Msg(string(debug.Stack()))
					log.Logger.Warn().
						Str("Input var name", varRawName).
						Str("Output var name", varName).
						Msgf(consts.MessageVarsSimplyfy, varName)
				}
				modifiers.Update = true
			} else {
				varName = varName + sfx
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Warn().
					Str("Input var name", varRawName).
					Str("Output var name", varName).
					Msg(consts.MessageVarsBadVarName)
				break Loop
			}
		case VarsSuffixes["Clear"]:
			if !modifiers.Clear &&
				parentVarModifiers == nil {
				modifiers.Clear = true
				if modifiers.Update {
					log.Logger.Debug().
						Msg(string(debug.Stack()))
					log.Logger.Warn().
						Str("Input var name", varRawName).
						Str("Output var name", varName).
						Msgf(consts.MessageVarsSimplyfy, varName)
				}
			} else {
				varName = varName + sfx
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Warn().
					Str("Input var name", varRawName).
					Str("Output var name", varName).
					Msg(consts.MessageVarsBadVarName)
				break Loop
			}
		case VarsSuffixes["Weak"]:
			if !(modifiers.Update ||
				modifiers.Weak) &&
				parentVarModifiers == nil {
				modifiers.Weak = true
			} else {
				varName = varName + sfx
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Warn().
					Str("Input var name", varRawName).
					Str("Output var name", varName).
					Msg(consts.MessageVarsBadVarName)
				break Loop
			}
		case VarsSuffixDelimeter:
			break Loop
		default:
			varName = varName + sfx
			break Loop
		}
	}
	return
}

// CombineVars func
func CombineVars(leftVars, rightVars *types.StackVars) (combinedVars *types.StackVars) {
	combinedVars = new(types.StackVars)
	combinedVars.Vars = make(map[string]interface{})
	combinedVars.Modifiers = make(map[string]interface{})

	mergedKeys := make(map[string]bool)
	for k := range leftVars.Vars {
		mergedKeys[k] = true
	}
	for k := range rightVars.Vars {
		mergedKeys[k] = true
	}

	for key := range mergedKeys {
		rightValue, ok := rightVars.Vars[key]
		rightKeyModifiers, ok := rightVars.Modifiers[key+thisVarModifiersSuffix].(StackVarsModifiers)
		if !ok {
			rightKeyModifiers = StackVarsModifiers{Weak: true}
		}
		leftValue, ok := leftVars.Vars[key]
		if !ok {
			combinedVars.Vars[key] = rightValue
			combinedVars.Modifiers[key] = rightVars.Modifiers[key]
			combinedVars.Modifiers[key+thisVarModifiersSuffix] = rightKeyModifiers
			continue
		}
		leftKeyModifiers, ok := leftVars.Modifiers[key+thisVarModifiersSuffix].(StackVarsModifiers)
		switch {
		case leftKeyModifiers.Weak:
			if rightKeyModifiers.Clear {
				combinedVars.Vars[key] = rightValue
				combinedVars.Modifiers[key] = rightVars.Modifiers[key]
				combinedVars.Modifiers[key+thisVarModifiersSuffix] = rightKeyModifiers
			} else {
				switch rightValue.(type) {
				case []interface{}:
					switch leftValue.(type) {
					case []interface{}:
						combinedVars.Vars[key] = append(leftValue.([]interface{}), rightValue.([]interface{})...)
						combinedVars.Modifiers[key+thisVarModifiersSuffix] = rightKeyModifiers
						combinedVars.Modifiers[key] = rightVars.Modifiers[key]
					default:
						combinedVars.Vars[key] = rightValue
						combinedVars.Modifiers[key+thisVarModifiersSuffix] = rightKeyModifiers
						combinedVars.Modifiers[key] = rightVars.Modifiers[key]
					}
				case map[string]interface{}:
					switch leftValue.(type) {
					case map[string]interface{}:
						newLeftVars := new(types.StackVars)
						switch leftVars.Modifiers[key].(type) {
						case map[string]interface{}:
							newLeftVars.Modifiers = leftVars.Modifiers[key].(map[string]interface{})
						}
						newLeftVars.Vars = leftValue.(map[string]interface{})
						newRightVars := new(types.StackVars)
						switch rightVars.Modifiers[key].(type) {
						case map[string]interface{}:
							newRightVars.Modifiers = rightVars.Modifiers[key].(map[string]interface{})
						}
						newRightVars.Vars = rightValue.(map[string]interface{})
						comboVars := CombineVars(newLeftVars, newRightVars)
						combinedVars.Vars[key] = comboVars.Vars
						combinedVars.Modifiers[key] = comboVars.Modifiers
						combinedVars.Modifiers[key+thisVarModifiersSuffix] = rightKeyModifiers
					default:
						combinedVars.Vars[key] = rightValue
						combinedVars.Modifiers[key] = rightVars.Modifiers[key]
						combinedVars.Modifiers[key+thisVarModifiersSuffix] = rightKeyModifiers
					}
				case nil:
					combinedVars.Vars[key] = leftValue
					combinedVars.Modifiers[key] = nil
					combinedVars.Modifiers[key+thisVarModifiersSuffix] = rightKeyModifiers
				default:
					combinedVars.Vars[key] = rightValue
					combinedVars.Modifiers[key] = nil
					combinedVars.Modifiers[key+thisVarModifiersSuffix] = rightKeyModifiers
				}
			}
		case leftKeyModifiers.Update:
			switch leftValue.(type) {
			case []interface{}:
				switch rightValue.(type) {
				case []interface{}:
					combinedVars.Vars[key] = append(rightValue.([]interface{}), leftValue.([]interface{})...)
					combinedVars.Modifiers[key+thisVarModifiersSuffix] = leftKeyModifiers
					combinedVars.Modifiers[key] = leftVars.Modifiers[key]
				default:
					combinedVars.Vars[key] = leftValue
					combinedVars.Modifiers[key+thisVarModifiersSuffix] = leftKeyModifiers
					combinedVars.Modifiers[key] = leftVars.Modifiers[key]
				}
			case map[string]interface{}:
				switch rightValue.(type) {
				case map[string]interface{}:
					newLeftVars := new(types.StackVars)
					newLeftVars.Modifiers = leftVars.Modifiers[key].(map[string]interface{})
					newLeftVars.Vars = leftValue.(map[string]interface{})
					newRightVars := new(types.StackVars)
					newRightVars.Modifiers, ok = rightVars.Modifiers[key].(map[string]interface{})
					newRightVars.Vars, ok = rightValue.(map[string]interface{})
					comboVars := CombineVars(newLeftVars, newRightVars)
					combinedVars.Vars[key] = comboVars.Vars
					combinedVars.Modifiers[key] = comboVars.Modifiers
					combinedVars.Modifiers[key+thisVarModifiersSuffix] = leftKeyModifiers
				default:
					combinedVars.Vars[key] = leftValue
					combinedVars.Modifiers[key] = leftVars.Modifiers[key]
					combinedVars.Modifiers[key+thisVarModifiersSuffix] = leftKeyModifiers
				}
			default:
				combinedVars.Vars[key] = leftValue
				combinedVars.Modifiers[key] = nil
				combinedVars.Modifiers[key+thisVarModifiersSuffix] = leftKeyModifiers
			}
		default:
			combinedVars.Vars[key] = leftValue
			combinedVars.Modifiers[key] = nil
			combinedVars.Modifiers[key+thisVarModifiersSuffix] = leftKeyModifiers
		}
	}
	return
}
