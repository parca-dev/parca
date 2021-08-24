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
  image: 'quay.io/parca/parca:dev',
  version: 'dev',
  replicas: 1,
  logLevel: 'debug',
  configPath: '/parca.yaml',
  corsAllowedOrigins: '*',
});

local parcaAgent = (import 'parca-agent/parca-agent.libsonnet')({
  name: 'parca-agent',
  namespace: ns.metadata.name,
  version: 'dev',
  image: 'quay.io/parca/parca-agent@sha256:a106b0b7fe5f5cc2c61120ef7f26577ebbdc6d62c410cbce546f1f280d736f0e',
  stores: ['%s.%s.svc.cluster.local:%d' % [parca.service.metadata.name, parca.service.metadata.namespace, parca.config.port]],
  logLevel: 'debug',
  insecure: true,
  insecureSkipVerify: true,
});

// Only for development purposes. Parca actually serves its UI itself.
local parcaUIDev = (import 'parca/parca-ui.libsonnet')({
  name: 'parca-ui',
  namespace: ns.metadata.name,
  image: 'quay.io/parca-dev/parca-ui:dev',
  version: 'dev',
  replicas: 1,
  apiEndpoint: 'http://localhost:7070',
  // apiEndpoint: 'http://%s.%s.svc.cluster.local:%d' % [parca.service.metadata.name, parca.service.metadata.namespace, parca.config.port],
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
} + {
  ['parca-ui-dev-' + name]: parcaUIDev[name]
  for name in std.objectFields(parcaUIDev)
  if parcaUIDev[name] != null
}
