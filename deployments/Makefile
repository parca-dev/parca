JSONNET_FMT := jsonnetfmt -n 2 --max-blank-lines 2 --string-style s --comment-style s

.PHONY: vendor
vendor:
	jb install

.PHONY: manifests
manifests: vendor $(shell find . -name 'vendor' -prune -o -name '*.libsonnet' -print -o -name '*.jsonnet' -print)
	rm -rf manifests
	mkdir manifests
	jsonnet -J vendor main.jsonnet -m manifests | xargs -I{} sh -c 'cat {} | gojsontoyaml > {}.yaml; rm -f {}' -- {}

fmt:
	find . -name 'vendor' -prune -o -name '*.libsonnet' -print -o -name '*.jsonnet' -print | \
		xargs -n 1 -- $(JSONNET_FMT) -i

