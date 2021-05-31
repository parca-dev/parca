local ns = {
  apiVersion: 'v1',
  kind: 'Namespace',
  metadata: {
    name: 'conprof',
  },
};

local bucketConfig = {
  bucketName: 'conprof',
  accessKey: 'minio',
  secretKey: 'minio123',
  endpoint: 'minio.%s.svc.cluster.local:9000' % ns.metadata.name,
  insecure: true,
};

local minio = (import 'minio/minio.libsonnet')({
  namespace: ns.metadata.name,
  bucketName: bucketConfig.bucketName,
  accessKey: bucketConfig.accessKey,
  secretKey: bucketConfig.secretKey,
});

local symbolicator = (import 'symbolicator/symbolicator.libsonnet') {
  local symbolicator = self,

  config+:: {
    name: 'symbolicator',
    namespace: ns.metadata.name,
    image: 'quay.io/conprof/symbolicator:v0.3.3',
    version: 'v0.3.3',
    rawconfig: {
      cache_dir: '/tmp/symbolicator',
      logging: {
        level: 'trace',
      },
      sources: [{
        id: 'minio',
        type: 's3',
        bucket: bucketConfig.bucketName,
        region: ['minio', 'http://%s' % bucketConfig.endpoint],
        access_key: bucketConfig.accessKey,
        secret_key: bucketConfig.secretKey,
        layout: { type: 'unified' },
        is_public: true,
      }],
    },
  },
};

local conprof = (import 'conprof/conprof.libsonnet') {
  local conprof = self,

  config+:: {
    name: 'conprof',
    namespace: ns.metadata.name,
    image: 'quay.io/brancz/conprof:ce33dfad53fb',
    version: '87e6b61b1feb',
    bucketConfig: bucketConfig,
    symbolServerURL: 'http://%s.%s.svc:3021/symbolicate' % [symbolicator.service.metadata.name, symbolicator.service.metadata.namespace],
  },
};

{
  '0kubenamespace': ns,
} + {
  ['conprof-' + name]: conprof[name]
  for name in std.objectFields(conprof)
  if conprof[name] != null
} + {
  ['symbolicator-' + name]: symbolicator[name]
  for name in std.objectFields(symbolicator)
  if symbolicator[name] != null
} + {
  ['minio-' + name]: minio[name]
  for name in std.objectFields(minio)
  if minio[name] != null
}
