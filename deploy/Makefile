JSONNET_FMT := jsonnetfmt -n 2 --max-blank-lines 2 --string-style s --comment-style s

.PHONY: vendor
vendor:
	jb install

.PHONY: manifests
manifests: vendor $(shell find . -name 'vendor' -prune -o -name '*.libsonnet' -print -o -name '*.jsonnet' -print)
	rm -rf manifests tilt
	mkdir manifests tilt
	jsonnet -J vendor main.jsonnet -m manifests | xargs -I{} sh -c 'cat {} | gojsontoyaml > {}.yaml; rm -f {}' -- {}
	# Using ls to make sure order preserved. ls sorts alphabetically.
	awk 'BEGINFILE {print "---"}{print}' `ls manifests/parca-agent*` > agent-manifest.yaml
	awk 'BEGINFILE {print "---"}{print}' `ls manifests/parca-server*` > server-manifest.yaml
	jsonnet -J vendor dev.jsonnet -m tilt | xargs -I{} sh -c 'cat {} | gojsontoyaml > {}.yaml; rm -f {}' -- {}

fmt:
	find . -name 'vendor' -prune -o -name '*.libsonnet' -print -o -name '*.jsonnet' -print | \
		xargs -n 1 -- $(JSONNET_FMT) -i
