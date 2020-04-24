local k = import 'ksonnet/ksonnet.beta.4/k.libsonnet';

(import 'conprof/conprof.libsonnet') {
  local conprof = self,

  config+:: {
    name:: 'conprof',
    namespace:: 'conprof',
    image:: 'quay.io/conprof/conprof:master-2020-04-24-66b9527',
    version:: 'master-2020-04-24-66b9527',
  },

  kubenamespace: k.core.v1.namespace.new(conprof.config.namespace),
}
