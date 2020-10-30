# stack tool ![stack tool][logo]

Stack is a Tool About Creating Kindda

- [stack tool !stack tool](#stack-tool-stack-tool)
  - [stack.yaml](#stackyaml)
    - [api](#api)
    - [libs](#libs)
    - [name](#name)
    - [vars](#vars)
    - [varsFrom](#varsfrom)
    - [flags](#flags)
    - [locals](#locals)
    - [run](#run)
    - [stacks](#stacks)
    - [when](#when)
    - [wait](#wait)
    - [waitGroups](#waitgroups)
  - [Exaples](#exaples)
  - [Used libraries](#used-libraries)
    - [google/cel-go](#googlecel-go)
    - [hairyhenderson/gomplate](#hairyhendersongomplate)
    - [google/go-jsonnet](#googlego-jsonnet)
    - [flosch/pongo2](#floschpongo2)
  - [Go project layout](#go-project-layout)
  - [Inspired by](#inspired-by)
  - [Thanx](#thanx)

## stack.yaml

```yaml
api: v1             # обязательный ключ.
libs: []            # необязательный ключ. список путей, в которых необходимо выполнять поиск стеков
name: stackDir      # обязательный ключ при inline пределении стека. Если стек определен через файл, то равно имени каталога с файлом stack.yaml
vars: {}            # необязательный ключ. словарь переменных
varsFrom: []        # необязательный ключ. список импорта в ключ vars
flags: {}           # необязательный ключ. словарь флагов доступных для использования в независимых стеках
locals: {}          # необязательный ключ. словарь локальных значений
preRun: []
run: []             # список команд для выполнения _последовательно_
stacks: []          # список стеков, для выполнения _последовательно_
pstacks: []         # список стеков, для выполнения _параллельно_
postRun: []
when: ""            # условие для выполнения стека (run && stacks)         _| See google/cel-go
wait: ""            # условие, которое стек будет ждать для своего запуска  | https://github.com/google/cel-go
waitGroups: []      # группы ожидания в которые входит текущий стек
# workdir:
```

### api

```yaml
api: v1
```

### libs

Типы библиотек:

1. Локальный каталог
2. git репозиторий

Порядок поиска локальных билиотек:

1. Каталог текущего стека
2. Библиотеки
3. Каталог корневого стека

```yaml
libs:
- libs
- git: https://gitlab.example.org/utility/tests.git
  ref: 5be7ad7861c8d39f60b7101fd8d8e816ff50353a
  path: libraries/tests
```

### name

Необходим только при inline определении стека

```yaml
name: stack_name
```

### vars

```yaml
# parent stack
vars:
  test1: value      # _затирает_ все последующие ключи в child stacks рекурсивно
  test2+: value     # _дополняет_ все последующие ключи test2 в child stacks ключами из test2
  test3++: value     # _дополняет_ все последующие ключи test3 в child stacks ключами из test3 _рекурсивно_
  test4~: value     # ТОЛЬКО КОРНЕВЫЕ КЛЮЧИ. слабый ключ (weak key) может быть затерт или дополнен соответствующими ключами в child stacks

# child stack
vars:
  test3-: value     # не может быть дополнен слабым ключем от родительского стека, может комбинироваться с другими модификаторами
  test5-^+: value   # символ ^ отделяет суффикс переменной от ее названия (test5-)
```

### varsFrom

```yaml
varsFrom:
- file: testVars.yaml
- sops: testSopsFile.yaml
```

### flags

```yaml
flags:   # значение может быть прочитано из любых стеков
  test1: value1
  test2: value2
  test3: value3
```

### locals

```yaml
locals:  # локальные ключи, актуальны только в текущем стеке
  test1: localvalue1
  test2: localvalue2
  test3: localvalue3
```

### run

```yaml
- gomplate: "{{ .name | filepath.Base }}"
  output:
  - strvar: namespace

- gomplate: |-
    {{ .vars.monitoring_grafana | toJSON }}
  output:
  - yml2var: monitoringGrafana

- gomplate:
  - templates/getEnvVars.gtpl
  output:
  - stderr

- jsonnet:
  - jsonnet/func.jsonnet
  output:
  - stderr

- jsonnet: |-
    function(stack)
      {test: stack.name}
  output:
  - stderr

- pongo2:
  - tpl/jinjaTemplate.jinja2
  output:
  - stdout

- script: scripts/example.sh
  output:
  - stdout

- group:
  - script: |-
      ping google.com -c 5
    output:
    - stderr
  - script: |-
      ping example.com -c 5
    output:
    - stderr
  parallel: true
  runTimeout: 10s
```

### stacks

```yaml
stacks:
- libs:
  - _base:
    - namespace
  - infra:
    - aws-node-termination-handler
    - aws-auth
    - cluster-autoscaler
    - eks-arm64
    - external-dns
    - chart: locals.chart_spec
```

### when

```yaml
when: vars.test1 == "value"
```

### wait

Стек будет ждать выполнения условия время заданное в wait_timeout (default: 5 min)

```yaml
wait: flags.test1 == "value1"
```

### waitGroups

```yaml
waitGroups:
- wg_example
- |- # cel may be used here
  "wg_" + name
```

---

## Exaples

[stack-examples](https://github.com/kruglovmax/stack-examples)
<!-- TODO -->
```yaml
```

---

## Used libraries

### google/cel-go

[docs](https://github.com/google/cel-spec/blob/master/doc/langdef.md),
[git](https://github.com/google/cel-go)

### hairyhenderson/gomplate

[docs](https://docs.gomplate.ca/),
[git](https://github.com/hairyhenderson/gomplate/)

### google/go-jsonnet

[docs](https://jsonnet.org/ref/language.html),
[git](https://github.com/google/go-jsonnet)

### flosch/pongo2

[docs](https://django.readthedocs.io/en/1.7.x/topics/templates.html),
[git](https://github.com/flosch/pongo2)

## Go project layout

<https://github.com/golang-standards/project-layout>

## Inspired by

[kapitan](https://github.com/deepmind/kapitan)\
[kasane](https://github.com/google/kasane)\
[argo-cd](https://github.com/argoproj/argo-cd)\
[fluxcd](https://github.com/fluxcd)

## Thanx

[![hd-deman](https://avatars1.githubusercontent.com/u/705532?s=30)](https://github.com/hd-deman)
[![pogossian](https://avatars1.githubusercontent.com/u/37933026?s=30)](https://github.com/pogossian)
[![zavgorodny](https://avatars1.githubusercontent.com/u/2486229?s=30)](https://github.com/zavgorodny)

<!-- DEFINITIONS -->

[logo]: https://github.com/kruglovmax/stack/raw/master/internal/stack30.png "logo"
