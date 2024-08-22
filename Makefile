.PHONY: debug no_debug fetch_dependencies

debug:
	go build -tags debug

no_debug:
	go build -ldflags="-s -w"

# for statusbar-right
fetch_dependencies:
	go get gopkg.in/yaml.v3


run-debug: debug
	./*

run: no_debug
	./*

