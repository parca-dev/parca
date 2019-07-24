local k = import 'ksonnet/ksonnet.beta.4/k.libsonnet';

(import 'conprof/conprof.libsonnet') {
  conprof+:: {
    namespace:
      k.core.v1.namespace.new($.conprof.config.namespace),
  },
}.conprof
