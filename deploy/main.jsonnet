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
  image: 'ghcr.io/parca-dev/parca:v0.0.3-alpha-11-g1f5b17c',
  version: 'v0.0.3-alpha',
  replicas: 1,
  logLevel: 'debug',
  configPath: '/parca.yaml',
  corsAllowedOrigins: '*',
});

local parcaAgent = (import 'parca-agent/parca-agent.libsonnet')({
  name: 'parca-agent',
  namespace: ns.metadata.name,
  version: 'v0.0.1-alpha',
  image: 'ghcr.io/parca-dev/parca-agent:v0.0.1-alpha.1',
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
