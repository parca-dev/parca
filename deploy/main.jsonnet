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
  image: 'ghcr.io/parca/parca:latest',
  version: 'latest',
  replicas: 1,
  logLevel: 'debug',
  configPath: '/parca.yaml',
  corsAllowedOrigins: '*',
});

local parcaAgent = (import 'parca-agent/parca-agent.libsonnet')({
  name: 'parca-agent',
  namespace: ns.metadata.name,
  version: 'latest',
  image: 'ghcr.io/parca/parca-agent:latest',
  stores: ['%s.%s.svc.cluster.local:%d' % [parca.service.metadata.name, parca.service.metadata.namespace, parca.config.port]],
  logLevel: 'debug',
  insecure: true,
  insecureSkipVerify: true,
  tempDir: 'tmp',
});

{
  'parca-agent-namespace': ns,
  'parca-server-namespace': ns,
} + {
  ['parca-server-' + name]: parca[name]
  for name in std.objectFields(parca)
  if parca[name] != null
} + {
  ['parca-agent-' + name]: parcaAgent[name]
  for name in std.objectFields(parcaAgent)
  if parcaAgent[name] != null
}
