local ns = {
  apiVersion: 'v1',
  kind: 'Namespace',
  metadata: {
    name: 'parca',
  },
};

local parca = (import 'parca/parca.libsonnet')({
  name: 'parca',
  namespace: ns.metadata.name,
  image: 'parca-dev/parca:dev',
  version: 'dev',
  replicas: 3,
  logLevel: 'debug',
});

local parcaAgent = (import 'parca-agent/parca-agent.libsonnet')({
  name: 'parca-agent',
  namespace: ns.metadata.name,
  version: 'dev',
  image: 'quay.io/parca/parca-agent@sha256:a106b0b7fe5f5cc2c61120ef7f26577ebbdc6d62c410cbce546f1f280d736f0e',
  stores: ['%s.%s.svc.cluster.local:9090' % [parca.service.metadata.name, parca.service.metadata.namespace]],
  insecure: true,
  insecureSkipVerify: true,
});

{
  '0namespace': ns,
} + {
  ['parca-' + name]: parca[name]
  for name in std.objectFields(parca)
  if parca[name] != null
} + {
  ['parca-agent-' + name]: parcaAgent[name]
  for name in std.objectFields(parcaAgent)
  if parcaAgent[name] != null
}
