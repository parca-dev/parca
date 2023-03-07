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

  config: {
    object_storage: {
      bucket: {
        type: 'FILESYSTEM',
        config: {
          directory: '/var/lib/parca',
        },
      },
    },
  },
  corsAllowedOrigins: '',
  logLevel: 'info',

  resources: {},
  port: 7070,

  serviceMonitor: false,
  livenessProbe: true,
  readinessProbe: true,
  storageRetentionTime: '',

  debugInfodUpstreamServers: ['https://debuginfod.systemtap.org'],
  debugInfodHTTPRequestTimeout: '5m',

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
  assert std.isBoolean(prc.config.livenessProbe),
  assert std.isBoolean(prc.config.readinessProbe),

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
          name: 'http',
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

  podSecurityPolicy: {
    apiVersion: 'policy/v1beta1',
    kind: 'PodSecurityPolicy',
    metadata: {
      name: prc.config.name,
    },
    spec: {
      allowPrivilegeEscalation: false,
      fsGroup: {
        ranges: [
          {
            max: 65535,
            min: 1,
          },
        ],
        rule: 'MustRunAs',
      },
      requiredDropCapabilities: [
        'ALL',
      ],
      runAsUser: {
        rule: 'MustRunAsNonRoot',
      },
      seLinux: {
        rule: 'RunAsAny',
      },
      supplementalGroups: {
        ranges: [
          {
            max: 65535,
            min: 1,
          },
        ],
        rule: 'MustRunAs',
      },
      volumes: [
        'configMap',
        'emptyDir',
        'projected',
        'secret',
        'downwardAPI',
        'persistentVolumeClaim',
      ],
    },
  },

  role: {
    apiVersion: 'rbac.authorization.k8s.io/v1',
    kind: 'Role',
    metadata: {
      name: prc.config.name,
      namespace: prc.config.namespace,
      labels: prc.config.commonLabels,
    },
    rules: [
      {
        apiGroups: [
          'policy',
        ],
        resourceNames: [
          prc.config.name,
        ],
        resources: [
          'podsecuritypolicies',
        ],
        verbs: [
          'use',
        ],
      },
    ],
  },

  roleBinding: {
    apiVersion: 'rbac.authorization.k8s.io/v1',
    kind: 'RoleBinding',
    metadata: {
      name: prc.config.name,
      namespace: prc.config.namespace,
      labels: prc.config.commonLabels,
    },
    roleRef: {
      apiGroup: 'rbac.authorization.k8s.io',
      kind: 'Role',
      name: prc.role.metadata.name,
    },
    subjects: [
      {
        kind: 'ServiceAccount',
        name: prc.serviceAccount.metadata.name,
      },
    ],
  },

  configMap: {
    apiVersion: 'v1',
    kind: 'ConfigMap',
    metadata: {
      name: prc.config.name,
      namespace: prc.config.namespace,
      labels: prc.config.commonLabels,
    },
    data: {
      'parca.yaml': std.manifestYamlDoc(prc.config.config),
    },
  },

  deployment:
    local c = {
      name: 'parca',
      image: prc.config.image,
      args:
        [
          '/parca',
          // http-address optionally specifies the TCP address for the server to listen on, in the form "host:port".
          '--http-address=' + ':' + prc.config.port,
          '--config-path=/etc/parca/parca.yaml',
          '--log-level=' + prc.config.logLevel,
        ] +
        (if prc.config.corsAllowedOrigins == '' then []
         else ['--cors-allowed-origins=' + prc.config.corsAllowedOrigins]) +
        (if prc.config.storageRetentionTime == '' then []
         else ['--storage-tsdb-retention-time=' + prc.config.storageRetentionTime]) +
        (if std.length(prc.config.debugInfodUpstreamServers) <= 0 then []
         else ['--debuginfod-upstream-servers=' + std.join(',', prc.config.debugInfodUpstreamServers)]) +
        (if prc.config.debugInfodHTTPRequestTimeout == '' then []
         else ['--debuginfod-http-request-timeout=' + prc.config.debugInfodHTTPRequestTimeout]),
      ports: [
        { name: port.name, containerPort: port.port }
        for port in prc.service.spec.ports
      ],
      volumeMounts: [
        {
          name: 'config',
          mountPath: '/etc/parca',
        },
      ] + (
        if prc.config.config.object_storage.bucket.type == 'FILESYSTEM' then [{
          name: 'data',
          mountPath: prc.config.config.object_storage.bucket.config.directory,
        }] else []
      ),
      resources: if prc.config.resources != {} then prc.config.resources else {},
      terminationMessagePolicy: 'FallbackToLogsOnError',
      livenessProbe: if prc.config.livenessProbe == true then {
        initialDelaySeconds: 5,
        exec: {
          command: ['/grpc_health_probe', '-v', '-addr=:' + prc.config.port],
        },
      },
      readinessProbe: if prc.config.readinessProbe == true then {
        initialDelaySeconds: 10,
        exec: {
          command: ['/grpc_health_probe', '-v', '-addr=:' + prc.config.port],
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
            volumes: [
              {
                name: 'config',
                configMap: { name: prc.configMap.metadata.name },
              },
            ] + (
              if prc.config.config.object_storage.bucket.type == 'FILESYSTEM' then [{
                name: 'data',
                emptyDir: {},
              }] else []
            ),
            nodeSelector: {
              'kubernetes.io/os': 'linux',
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
          port: prc.service.spec.ports[0].name,
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
