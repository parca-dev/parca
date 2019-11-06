local k = import 'ksonnet/ksonnet.beta.4/k.libsonnet';

(import 'conprof/conprof.libsonnet') {
  name:: 'conprof',
  namespace:: 'conprof',
  image:: 'quay.io/brancz/conprof:v0.1.0-dev',
  version:: 'v0.1.0-dev',

  statefulset+: {
    spec+: {
      template+: {
        spec+: {
          containers:
            std.map(
              function(c) c { imagePullPolicy: 'Always', args+: ['--log.level=debug'] },
              super.containers,
            ),
        },
      },
    },
  },
  kubenamespace: k.core.v1.namespace.new($.namespace),
}
