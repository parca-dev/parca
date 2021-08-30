// These are the defaults for this components configuration.
// When calling the function to generate the component's manifest,
// you can pass an object structured like the default to overwrite default values.
local defaults = {
  local defaults = self,
  name: 'parca',
  namespace: error 'must provide namespace',
  version: error 'must provide version',
  image: error 'must provide image',
  replicas: error 'must provide replicas',

  configPath: 'parca.yaml',
  corsAllowedOrigins: '',
  logLevel: 'info',

  resources: {},
  port: 7070,

  serviceMonitor: false,

  commonLabels:: {
    'app.kubernetes.io/name': 'parca',
    'app.kubernetes.io/instance': defaults.name,
    'app.kubernetes.io/version': defaults.version,
    'app.kubernetes.io/component': 'observability',
  },

  podLabelSelector:: {
    [labelName]: defaults.commonLabels[labelName]
    for labelName in std.objectFields(defaults.commonLabels)
    if labelName != 'app.kubernetes.io/version'
  },

  securityContext:: {
    fsGroup: 65534,
    runAsUser: 65534,
  },
};

function(params) {
  local prc = self,

  // Combine the defaults and the passed params to make the component's config.
  config:: defaults + params,
  // Safety checks for combined config of defaults and params
  assert std.isNumber(prc.config.replicas) && prc.config.replicas >= 0 : 'parca replicas has to be number >= 0',
  assert std.isObject(prc.config.resources),
  assert std.isBoolean(prc.config.serviceMonitor),

  service: {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: {
      name: prc.config.name,
      namespace: prc.config.namespace,
      labels: prc.config.commonLabels,
    },
    spec: {
      ports: [
        {
          assert std.isNumber(prc.config.port),

          name: 'all',
          port: prc.config.port,
          targetPort: prc.config.port,
        },
      ],
      selector: prc.config.podLabelSelector,
    },
  },

  serviceAccount: {
    apiVersion: 'v1',
    kind: 'ServiceAccount',
    metadata: {
      name: prc.config.name,
      namespace: prc.config.namespace,
      labels: prc.config.commonLabels,
    },
  },

  deployment:
    local c = {
      name: 'parca',
      image: prc.config.image,
      args:
        [
          '/parca',
          '--config-path=' + prc.config.configPath,
          '--log-level=' + prc.config.logLevel,
        ] +
        (if prc.config.corsAllowedOrigins != '' then []
         else ['--cors-allowed-origins=' + prc.config.corsAllowedOrigins]),
      ports: [
        { name: port.name, containerPort: port.port }
        for port in prc.service.spec.ports
      ],
      resources: if prc.config.resources != {} then prc.config.resources else {},
      terminationMessagePolicy: 'FallbackToLogsOnError',
      livenessProbe: {
        initialDelaySeconds: 5,
        exec: {
          command: ['/grpc-health-probe', '-v', '-addr=:' + prc.config.port],
        },
      },
      readinessProbe: {
        initialDelaySeconds: 10,
        exec: {
          command: ['/grpc-health-probe', '-v', '-addr=:' + prc.config.port],
        },
      },
    };

    {
      apiVersion: 'apps/v1',
      kind: 'Deployment',
      metadata: {
        name: prc.config.name,
        namespace: prc.config.namespace,
        labels: prc.config.commonLabels,
      },
      spec: {
        replicas: prc.config.replicas,
        selector: { matchLabels: prc.config.podLabelSelector },
        template: {
          metadata: {
            labels: prc.config.commonLabels,
          },
          spec: {
            containers: [c],
            securityContext: prc.config.securityContext,
            serviceAccountName: prc.serviceAccount.metadata.name,
            terminationGracePeriodSeconds: 120,
            nodeSelector: {
              'beta.kubernetes.io/os': 'linux',
            },
          },
        },
      },
    },

  serviceMonitor: if prc.config.serviceMonitor == true then {
    apiVersion: 'monitoring.coreos.com/v1',
    kind: 'ServiceMonitor',
    metadata+: {
      name: prc.config.name,
      namespace: prc.config.namespace,
      labels: prc.config.commonLabels,
    },
    spec: {
      selector: {
        matchLabels: prc.config.podLabelSelector,
      },
      endpoints: [
        {
          port: 'http',
          relabelings: [{
            sourceLabels: ['namespace', 'pod'],
            separator: '/',
            targetLabel: 'instance',
          }],
        },
      ],
    },
  },
}
