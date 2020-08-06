package misc

import (
	"encoding/json"
	"io/ioutil"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/kruglovmax/stack/pkg/log"
	sopsDecrypt "go.mozilla.org/sops/v3/decrypt"
	"sigs.k8s.io/yaml"
)

// LoadYAML func
func LoadYAML(yamlInput string, result interface{}) {
	var yamlErr error
	yamlErr = yaml.Unmarshal([]byte(yamlInput), &result)
	if yamlErr != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(yamlInput))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(yamlErr.Error())
	}
}

// LoadYAMLFromFile func
func LoadYAMLFromFile(fileName string, result interface{}) {
	content, ioErr := ioutil.ReadFile(fileName)
	if ioErr != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(fileName))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(ioErr.Error())
	}
	LoadYAML(string(content), &result)
}

// LoadYAMLFromSopsFile func
func LoadYAMLFromSopsFile(fileName string, result interface{}) {
	content, err := sopsDecrypt.File(fileName, "yaml")
	if err != nil {
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msgf("[ SOPS ] File %s decryption Error. Check internet connection.\n"+err.Error(), fileName)
	}
	LoadYAML(string(content), &result)
}

// ToYAML func
func ToYAML(object interface{}) string {
	y, err := yaml.Marshal(object)
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(object))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("YAML marshal error\n" + err.Error())
	}
	return string(y)
}

// ToJSON func
func ToJSON(object interface{}) string {
	y, err := json.Marshal(object)
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(object))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg("JSON marshal error\n" + err.Error())
	}
	return string(y)
}

// GetRunItemOutputType func
func GetRunItemOutputType(item interface{}) []interface{} {
	result, ok := (item).(map[string]interface{})["output"].([]interface{})
	if ok {
		return result
	}
	return nil
}

// GetObjectValue func
func GetObjectValue(object interface{}, key string) interface{} {
	switch object.(type) {
	case map[string]interface{}:
		return object.(map[string]interface{})[key]
	case []interface{}:
		i, err := strconv.Atoi(key)
		if err != nil {
			log.Logger.Trace().
				Msg(spew.Sdump(key))
			log.Logger.Debug().
				Msg(string(debug.Stack()))
			log.Logger.Fatal().Str("Index", key).
				Msg(err.Error())
		}
		return object.([]interface{})[i]
	default:
		log.Logger.Warn().
			Msg("Bad value\n" + string(debug.Stack()))
		return nil
	}
}

// GetObject func
func GetObject(root interface{}, path string) interface{} {
	splitResult := strings.SplitN(strings.TrimLeft(path, "."), ".", 2)
	left := splitResult[0]
	if len(splitResult) > 0 && left != "" {
		if len(splitResult) > 1 {
			right := splitResult[1]
			return GetObject(GetObjectValue(root, left), right)
		}
		return GetObjectValue(root, left)
	}
	return root
}

// UniqueStrings Returns unique string items in a slice
// TODO FIX BUG HERE disable items sorting
func UniqueStrings(slice []string) []string {
	// create a map with all the values as key
	uniqMap := make(map[string]struct{})
	for _, v := range slice {
		uniqMap[v] = struct{}{}
	}

	// turn the map keys into a slice
	uniqSlice := make([]string, 0, len(uniqMap))
	for v := range uniqMap {
		uniqSlice = append(uniqSlice, v)
	}
	return uniqSlice
}

// GetDirsByRegexp func
func GetDirsByRegexp(pwd string, pattern string) (dirs []string) {
	// ^(?:monitoring|(.*))$
	pattern = "^(" + pattern + ")$"
	files, err := ioutil.ReadDir(pwd)
	if err != nil {
		dirs = nil
		return
	}
	for _, fileItem := range files {
		if fileItem.IsDir() {
			re, err := regexp.Compile(pattern)
			if err != nil {
				log.Logger.Fatal().
					Msg(err.Error() + "\n" + string(debug.Stack()))
			}
			matches := re.FindStringSubmatch(fileItem.Name())
			if len(matches) > 0 && matches[1] != "" {
				dirs = append(dirs, fileItem.Name())
			}
		}
	}
	return
}

// TagsMatcher func
func TagsMatcher(tags, patterns []string) (result bool) {
	// if tags == nil {
	// 	result = true
	// 	return
	// }
	matches := make(map[string]bool)
	result = true
	for _, tag := range tags {
		result = false
		for _, pattern := range patterns {
			matched, err := regexp.Match("^("+pattern+")$", []byte(tag))
			if err != nil {
				log.Logger.Trace().
					Msg(spew.Sdump(pattern, tag))
				log.Logger.Debug().
					Msg(string(debug.Stack()))
				log.Logger.Fatal().
					Msg(err.Error())
			}
			if matched {
				matches[pattern] = true
				result = true
				break
			}
		}
		if len(matches) == len(patterns) {
			break
		}
	}
	return
}
