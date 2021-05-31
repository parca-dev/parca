{
  local symbolicator = self,

  config:: {
    name: error 'must provide name',
    namespace: error 'must provide namespace',
    image: error 'must provide image',
    version: error 'must set version',

    commonLabels:: {
      'app.kubernetes.io/name': 'symbolicator',
      'app.kubernetes.io/instance': symbolicator.config.name,
    },

    commonLabelsWithVersion:: {
      'app.kubernetes.io/name': 'symbolicator',
      'app.kubernetes.io/instance': symbolicator.config.name,
      'app.kubernetes.io/version': symbolicator.config.version,
    },

    rawconfig:: {
    },
  },

  secret: {
    apiVersion: 'v1',
    kind: 'Secret',
    metadata: {
      name: symbolicator.config.name,
      namespace: symbolicator.config.namespace,
      labels: symbolicator.config.commonLabels,
    },
    stringData: {
      'config.yaml': std.manifestYamlDoc(symbolicator.config.rawconfig),
    },
    type: 'Opaque',
  },

  deployment: {
    apiVersion: 'apps/v1',
    kind: 'Deployment',
    metadata: {
      name: symbolicator.config.name,
      namespace: symbolicator.config.namespace,
      labels: symbolicator.config.commonLabels,
    },
    spec: {
      selector: {
        matchLabels: symbolicator.config.commonLabels,
      },
      strategy: {
        type: 'Recreate',
      },
      template: {
        metadata: {
          labels: symbolicator.config.commonLabels,
        },
        spec: {
          serviceAccountName: symbolicator.serviceAccount.metadata.name,
          containers: [
            {
              args: [
                'run',
                '--config=/etc/symbolicator/config.yaml',
              ],
              image: symbolicator.config.image,
              name: 'symbolicator',
              ports: [
                {
                  containerPort: 3021,
                },
              ],
              volumeMounts: [
                {
                  mountPath: '/etc/symbolicator',
                  name: 'config',
                  readOnly: true,
                },
                {
                  mountPath: '/tmp/symbolicator',
                  name: 'cache',
                  readOnly: false,
                },
              ],
            },
          ],
          volumes: [
            {
              name: 'config',
              secret: {
                secretName: symbolicator.secret.metadata.name,
              },
            },
            {
              emptyDir: {},
              name: 'cache',
            },
          ],
        },
      },
    },
  },

  serviceAccount: {
    apiVersion: 'v1',
    kind: 'ServiceAccount',
    metadata: {
      name: symbolicator.config.name,
      namespace: symbolicator.config.namespace,
      labels: symbolicator.config.commonLabels,
    },
  },

  service: {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: {
      name: symbolicator.config.name,
      namespace: symbolicator.config.namespace,
      labels: symbolicator.config.commonLabels,
    },
    spec: {
      ports: [
        {
          port: 3021,
          protocol: 'TCP',
          targetPort: 3021,
        },
      ],
      selector: symbolicator.config.commonLabels,
      type: 'ClusterIP',
    },
  },
}
