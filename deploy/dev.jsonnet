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
  image: 'quay.io/parca/parca-agent:dev',
  stores: ['%s.%s.svc.cluster.local:%d' % [parca.service.metadata.name, parca.service.metadata.namespace, parca.config.port]],
  logLevel: 'debug',
  insecure: true,
  insecureSkipVerify: true,
  tempDir: 'tmp',
});

// Only for development purposes. Parca actually serves its UI itself.
local parcaUI = (import 'parca/parca-ui.libsonnet')({
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
  ['parca-server-' + name]: parca[name]
  for name in std.objectFields(parca)
  if parca[name] != null
} + {
  ['parca-agent-' + name]: parcaAgent[name]
  for name in std.objectFields(parcaAgent)
  if parcaAgent[name] != null
} + {
  ['parca-ui-' + name]: parcaUI[name]
  for name in std.objectFields(parcaUI)
  if parcaUI[name] != null
}
