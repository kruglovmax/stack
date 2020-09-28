package schema

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
  waitGroups:
    type: array
    uniqueItems: true
    items:
      type: string
      minLength: 1
  outputType:
    array:
    uniqueItems: true
    items:
      oneOf:
      - enum:
        - stdout
        - stderr
        - 'yml2var: var'
        - 'str2var: var'
      - type: object
        additionalProperties: false
        minProperties: 1
        maxProperties: 1
        properties:
          yml2var:
            type: string
            minLength: 1
          str2var:
            type: string
            minLength: 1
  runItemVars:
    oneOf:
    - enum:
      - 'var name in .vars (see documentation)'
      - 'yaml map (see documentation)'
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
    - type: array
      items: { "$ref": "#/definitions/subStackWithChilds" }
  run:
    type: array
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
          when: { "$ref": "#/definitions/when" }
          wait: { "$ref": "#/definitions/when" }
          runTimeout: { "$ref": "#/definitions/timeout" }
          waitTimeout: { "$ref": "#/definitions/timeout" }
      - type: object
        additionalProperties: false
        minProperties: 1
        required: ["jsonnet", "output"]
        properties:
          jsonnet:
            oneOf:
            - type: string
              minLength: 0
            - type: array
              uniqueItems: true
              minItems: 1
              maxItems: 1
              items:
                type: string
                minLength: 1
          vars: { "$ref": "#/definitions/runItemVars" }
          output: { "$ref": "#/definitions/outputType" }
          when: { "$ref": "#/definitions/when" }
          wait: { "$ref": "#/definitions/when" }
          runTimeout: { "$ref": "#/definitions/timeout" }
          waitTimeout: { "$ref": "#/definitions/timeout" }
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
          when: { "$ref": "#/definitions/when" }
          wait: { "$ref": "#/definitions/when" }
          runTimeout: { "$ref": "#/definitions/timeout" }
          waitTimeout: { "$ref": "#/definitions/timeout" }
      - type: object
        additionalProperties: false
        minProperties: 1
        required: ["script"]
        properties:
          script:
            type: string
          vars: { "$ref": "#/definitions/runItemVars" }
          output: { "$ref": "#/definitions/outputType" }
          when: { "$ref": "#/definitions/when" }
          wait: { "$ref": "#/definitions/when" }
          runTimeout: { "$ref": "#/definitions/timeout" }
          waitTimeout: { "$ref": "#/definitions/timeout" }
      - type: object
        additionalProperties: false
        minProperties: 1
        required: ["gitclone"]
        properties:
          gitclone:
            type: string
          ref:
            type: string
            minLength: 1
          dir:
            type: string
            minLength: 1
          when: { "$ref": "#/definitions/when" }
          wait: { "$ref": "#/definitions/when" }
          runTimeout: { "$ref": "#/definitions/timeout" }
          waitTimeout: { "$ref": "#/definitions/timeout" }
      - type: object
        additionalProperties: false
        minProperties: 1
        required: ["group"]
        properties:
          group: { "$ref": "#/definitions/run" }
          when: { "$ref": "#/definitions/when" }
          wait: { "$ref": "#/definitions/when" }
          runTimeout: { "$ref": "#/definitions/timeout" }
          waitTimeout: { "$ref": "#/definitions/timeout" }
          parallel:
            type: boolean
      #- oneOf:
      #  - type: object
      #    additionalProperties: false
      #    required: ["chart", "repo", "name", "namespace", "output"]
      #    properties:
      #      chart:
      #        type: string
      #        minLength: 1
      #        format: hostname
      #      name:
      #        type: string
      #        minLength: 1
      #        format: hostname
      #      namespace:
      #        type: string
      #        minLength: 1
      #        format: hostname
      #      vars: { "$ref": "#/definitions/runItemVars" }
      #      repo:
      #        type: string
      #        # url
      #        pattern: ^(http|https)\://([a-zA-Z0-9\.\-]+(\:[a-zA-Z0-9\.&amp;%\$\-]+)*@)*((25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[1-9])\.(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[1-9]|0)\.(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[1-9]|0)\.(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[0-9])|localhost|([a-zA-Z0-9\-]+\.)*[a-zA-Z0-9\-]+\.(com|edu|gov|int|mil|net|org|biz|arpa|info|name|pro|aero|coop|museum|[a-zA-Z]{2}))(\:[0-9]+)*(/($|[a-zA-Z0-9\.\,\?\'\\\+&amp;%\$#\=~_\-]+))*$
      #      version:
      #        type: string
      #        # semver
      #        pattern: ^([0-9]+)\.([0-9]+)\.([0-9]+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+)?$
      #      output: { "$ref": "#/definitions/outputType" }
      #  - type: object
      #    additionalProperties: false
      #    required: ["chart", "name", "namespace", "output"]
      #    properties:
      #      chart:
      #        type: string
      #        minLength: 1
      #      name:
      #        type: string
      #        minLength: 1
      #        format: hostname
      #      namespace:
      #        type: string
      #        minLength: 1
      #        format: hostname
      #      vars: { "$ref": "#/definitions/runItemVars" }
      #      output: { "$ref": "#/definitions/outputType" }
  api:
    type: string
    enum:
    - v1
  name:
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
      ref:
        type: string
        minLength: 1
      path:
        type: string
        minLength: 1
  when:
    type: string
    minLength: 1
  timeout:
    type: string
    minLength: 2
  stack:
    type: object
    additionalProperties: false
    properties:
      api: { "$ref": "#/definitions/api" }
      name: { "$ref": "#/definitions/name" }
      vars: { "$ref": "#/definitions/vars" }
      flags: { "$ref": "#/definitions/vars" }
      locals: { "$ref": "#/definitions/vars" }
      varsFrom: { "$ref": "#/definitions/varsFrom" }
      preRun: { "$ref": "#/definitions/run" }
      run: { "$ref": "#/definitions/run" }
      postRun: { "$ref": "#/definitions/run" }
      stacks: { "$ref": "#/definitions/stacks" }
      libs: { "$ref": "#/definitions/libs" }
      when: { "$ref": "#/definitions/when" }
      wait: { "$ref": "#/definitions/when" }
      waitGroups: { "$ref": "#/definitions/waitGroups" }
      waitTimeout: { "$ref": "#/definitions/timeout" }


allOf:
- "$ref": "#/definitions/stack"
- required: ["api"]
`
)

// ConfigSchema var
var ConfigSchema *jsonschema.Schema

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

func init() {
	ConfigSchema = mustCompileConfigSchema()
}
