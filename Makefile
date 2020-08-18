.DEFAULT: all
.PHONY: all

CURRENT_OS=$(shell go env GOOS)
CURRENT_OS_ARCH=$(shell go env GOARCH)
GOBIN?=$(shell echo `go env GOPATH`/bin)

MAIN_GO_MODULE:=$(shell go list -m -f '{{ .Path }}')
LOCAL_GO_MODULES:=$(shell go list -m -f '{{ .Path }}' all | grep $(MAIN_GO_MODULE))
godeps=$(shell go list -deps -f '{{if not .Standard}}{{ $$dep := . }}{{range .GoFiles}}{{$$dep.Dir}}/{{.}} {{end}}{{end}}' $(1) | sed "s%${PWD}/%%g")


# NB default target architecture is amd64. If you would like to try the
# other one -- pass an ARCH variable, e.g.,
#  `make ARCH=arm64`
ifeq ($(ARCH),)
	ARCH=${CURRENT_OS_ARCH}
endif

ifeq ($(OS),)
	OS=${CURRENT_OS}
endif

all: build/stack

## help: show help message
help: Makefile
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'

## build/stack: build stack tool
build/stack: \
		clean/stack \
		cmd/stack/*.go \
		pkg/*/*
	CGO_ENABLED=0 GOOS=${OS} GOARCH=${ARCH} go build -ldflags="-s -w" -o $@ ./cmd/stack
ifeq (${OS}, linux)
	# upx --brute $@
	# upx $@
endif

ifeq (${USER}, maxim)
	cp build/stack ~/work/scripts/stack3
endif

## clean/stack: rm build/stack
clean/stack:
ifneq (,$(wildcard build/stack))
	rm build/stack
endif
