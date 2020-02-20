local k = import 'ksonnet/ksonnet.beta.4/k.libsonnet';

(import 'main.jsonnet') {
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
}
