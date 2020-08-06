package v1alpha1

import (
	"runtime/debug"

	"github.com/davecgh/go-spew/spew"
	"github.com/kruglovmax/stack/pkg/log"
	jsonschema "github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"
)

const (
	configSchemaYAML = `
$schema: http://json-schema.org/draft/2019-09/schema#

definitions:
  outputType:
    array:
    uniqueItems: true
    items:
      anyOf:
      - const: stdout
      - const: stderr
      - type: object
        additionalProperties: false
        minProperties: 1
        maxProperties: 1
        properties:
          ymlvar:
            type: string
            minLength: 1
          strvar:
            type: string
            minLength: 1
  runItemVars:
    oneOf:
    - type: string
      minLength: 1
    - type: object
  stacks:
    type: array
    uniqueItems: true
    items:
      anyOf:
      - type: string
        minLength: 1
      - allOf:
        - { "$ref": "#/definitions/stack" }
        - required: ["name"]
        - anyOf:
          - required: ["run"]
          - required: ["stacks"]
        - minProperties: 2
      - { "$ref": "#/definitions/subStackWithChilds" }
  subStackWithChilds:
    anyOf:
    - type: object
      additionalProperties: false
      minProperties: 1
      patternProperties:
        ".*":
          anyOf:
          - type: array
            uniqueItems: true
            items:
              anyOf:
              - { "$ref": "#/definitions/subStackWithChilds" }
              - type: string
                minLength: 1
          - type: object
            additionalProperties: false
            minProperties: 1
            properties:
              vars: { "$ref": "#/definitions/vars" }
              tags: { "$ref": "#/definitions/tags" }
              workspace: { "$ref": "#/definitions/workspace" }
    - type: array
      items: { "$ref": "#/definitions/subStackWithChilds" }
  run:
    oneOf:
    - type: array
      items:
        anyOf:
        - type: object
          additionalProperties: false
          minProperties: 1
          required: ["gomplate", "output"]
          properties:
            gomplate:
              oneOf:
              - type: string
                minLength: 0
              - type: array
                uniqueItems: true
                items:
                  type: string
                  minLength: 1
            vars: { "$ref": "#/definitions/runItemVars" }
            output: { "$ref": "#/definitions/outputType" }
        - type: object
          additionalProperties: false
          minProperties: 1
          required: ["pongo2", "output"]
          properties:
            pongo2:
              oneOf:
              - type: string
                minLength: 1
              - type: array
                uniqueItems: true
                items:
                  type: string
                  minLength: 1
            vars: { "$ref": "#/definitions/runItemVars" }
            output: { "$ref": "#/definitions/outputType" }
        - oneOf:
          - type: object
            additionalProperties: false
            required: ["chart", "repo", "name", "namespace", "output"]
            properties:
              chart:
                type: string
                minLength: 1
                format: hostname
              name:
                type: string
                minLength: 1
                format: hostname
              namespace:
                type: string
                minLength: 1
                format: hostname
              vars: { "$ref": "#/definitions/runItemVars" }
              repo:
                type: string
                # url
                pattern: ^(http|https)\://([a-zA-Z0-9\.\-]+(\:[a-zA-Z0-9\.&amp;%\$\-]+)*@)*((25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[1-9])\.(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[1-9]|0)\.(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[1-9]|0)\.(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[0-9])|localhost|([a-zA-Z0-9\-]+\.)*[a-zA-Z0-9\-]+\.(com|edu|gov|int|mil|net|org|biz|arpa|info|name|pro|aero|coop|museum|[a-zA-Z]{2}))(\:[0-9]+)*(/($|[a-zA-Z0-9\.\,\?\'\\\+&amp;%\$#\=~_\-]+))*$
              version:
                type: string
                # semver
                pattern: ^([0-9]+)\.([0-9]+)\.([0-9]+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+)?$
              output: { "$ref": "#/definitions/outputType" }
          - type: object
            additionalProperties: false
            required: ["chart", "name", "namespace", "output"]
            properties:
              chart:
                type: string
                minLength: 1
              name:
                type: string
                minLength: 1
                format: hostname
              namespace:
                type: string
                minLength: 1
                format: hostname
              vars: { "$ref": "#/definitions/runItemVars" }
              output: { "$ref": "#/definitions/outputType" }
        - type: object
          additionalProperties: false
          minProperties: 1
          required: ["script"]
          properties:
            script:
              type: string
            vars: { "$ref": "#/definitions/runItemVars" }
            output: { "$ref": "#/definitions/outputType" }
            timeout:
              type: integer
              minimum: 0
  api:
    oneOf:
    - const: v1alpha1
  name:
    type: string
    minLength: 1
  workspace:
    type: string
    minLength: 1
  vars:
    type: object
  varsFrom:
    type: array
    items:
      anyOf:
      - type: object
        properties:
          file:
            type: string
            minLength: 1
      - type: object
        properties:
          sops:
            type: string
            minLength: 1
  tags:
    type: array
    uniqueItems: true
    items:
      type: string
      minLength: 1
  message:
    type: string
  libs:
    oneOf:
    - type: string
      minLength: 1
    - type: array
      uniqueItems: true
      items:
        anyOf:
        - type: string
          minLength: 1
        - { "$ref": "#/definitions/gitdir" }
  gitdir:
    type: object
    additionalProperties: false
    properties:
      git:
        type: string
        minLength: 3
      commit:
        type: string
        minLength: 1
      path:
        type: string
        minLength: 1
  conditions:
    type: array
    uniqueItems: true
    items:
      type: string
      minLength: 1
  stack:
    type: object
    additionalProperties: false
    properties:
      api: { "$ref": "#/definitions/api" }
      name: { "$ref": "#/definitions/name" }
      workspace: { "$ref": "#/definitions/workspace" }
      vars: { "$ref": "#/definitions/vars" }
      varsFrom: { "$ref": "#/definitions/varsFrom" }
      run: { "$ref": "#/definitions/run" }
      stacks: { "$ref": "#/definitions/stacks" }
      tags: { "$ref": "#/definitions/tags" }
      message: { "$ref": "#/definitions/message" }
      libs: { "$ref": "#/definitions/libs" }
      conditions: { "$ref": "#/definitions/conditions" }


allOf:
- "$ref": "#/definitions/stack"
- required: ["api"]
`
)

func mustCompileConfigSchema() *jsonschema.Schema {
	j, err := yaml.YAMLToJSON([]byte(configSchemaYAML))
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(configSchemaYAML))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(err.Error())
	}
	sl := jsonschema.NewSchemaLoader()
	sl.Validate = false
	schema, err := sl.Compile(jsonschema.NewBytesLoader(j))
	if err != nil {
		log.Logger.Trace().
			Msg(spew.Sdump(j))
		log.Logger.Debug().
			Msg(string(debug.Stack()))
		log.Logger.Fatal().
			Msg(err.Error())
	}
	return schema
}

// ConfigSchema var
var ConfigSchema = mustCompileConfigSchema()
