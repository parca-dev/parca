local k = import 'ksonnet/ksonnet.beta.4/k.libsonnet';

(import 'conprof/conprof.libsonnet') {
  local conprof = self,

  config+:: {
    name:: 'conprof',
    namespace:: 'conprof',
    image:: 'quay.io/brancz/conprof:v0.1.0-dev',
    version:: 'v0.1.0-dev',
  },

  kubenamespace: k.core.v1.namespace.new(conprof.config.namespace),
}
