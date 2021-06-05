local ns = {
  apiVersion: 'v1',
  kind: 'Namespace',
  metadata: {
    name: 'conprof',
  },
};

local conprof = (import 'conprof/conprof.libsonnet') {
  local conprof = self,

  config+:: {
    name: 'conprof',
    namespace: ns.metadata.name,
    image: 'quay.io/brancz/conprof:08a162ee22cd',
    version: '87e6b61b1feb',
    bucketConfig: {
      type: 'FILESYSTEM',
      config: {
        directory: '/tmp',
      },
    },
  },
};

{
  '0kubenamespace': ns,
} + {
  ['conprof-' + name]: conprof[name]
  for name in std.objectFields(conprof)
  if conprof[name] != null
}
