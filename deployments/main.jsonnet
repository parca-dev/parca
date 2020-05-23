local k = import 'ksonnet/ksonnet.beta.4/k.libsonnet';

(import 'conprof/conprof.libsonnet') {
  local conprof = self,

  config+:: {
    name:: 'conprof',
    namespace:: 'conprof',
    image:: 'quay.io/conprof/conprof:master-2020-05-20-8e0ac0f',
    version:: 'master-2020-05-20-8e0ac0f',
  },

  kubenamespace: k.core.v1.namespace.new(conprof.config.namespace),
}
