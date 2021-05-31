local k3 = import 'ksonnet/ksonnet.beta.3/k.libsonnet';
local k = import 'ksonnet/ksonnet.beta.4/k.libsonnet';

{
  local conprof = self,

  config:: {
    name: error 'must provide name',
    namespace: error 'must provide namespace',
    image: error 'must provide image',
    version: error 'must set version',
    namespaces: [conprof.config.namespace],

    symbolServerURL: '',
    bucketConfig: null,

    commonLabels:: {
      'app.kubernetes.io/name': 'conprof',
      'app.kubernetes.io/instance': conprof.config.name,
      'app.kubernetes.io/version': conprof.config.version,
    },

    podLabelSelector:: {
      [labelName]: conprof.config.commonLabels[labelName]
      for labelName in std.objectFields(conprof.config.commonLabels)
      if !std.setMember(labelName, ['app.kubernetes.io/version'])
    },

    rawconfig:: {
      scrape_configs: [{
        job_name: 'conprof',
        kubernetes_sd_configs: [{
          namespaces: { names: conprof.config.namespaces },
          role: 'pod',
        }],
        relabel_configs: [
          {
            action: 'keep',
            regex: 'conprof.*',
            source_labels: ['__meta_kubernetes_pod_name'],
          },
          {
            source_labels: ['__meta_kubernetes_namespace'],
            target_label: 'namespace',
          },
          {
            source_labels: ['__meta_kubernetes_pod_name'],
            target_label: 'pod',
          },
          {
            source_labels: ['__meta_kubernetes_pod_container_name'],
            target_label: 'container',
          },
        ],
        scrape_interval: '1m',
        scrape_timeout: '1m',
      }],
    },
  },

  roleBindings: {
    apiVersion: 'rbac.authorization.k8s.io/v1',
    kind: 'RoleBindingList',
    items: [
      {
        apiVersion: 'rbac.authorization.k8s.io/v1',
        kind: 'RoleBinding',
        metadata: {
          labels: conprof.config.commonLabels,
          name: conprof.config.name,
          namespace: conprof.config.namespace,
        },
        roleRef: {
          apiGroup: 'rbac.authorization.k8s.io',
          kind: 'Role',
          name: conprof.config.name,
        },
        subjects: [
          {
            kind: 'ServiceAccount',
            name: conprof.serviceAccount.metadata.name,
            namespace: namespace,
          },
        ],
      }
      for namespace in conprof.config.namespaces
    ],
  },

  roles: {
    apiVersion: 'rbac.authorization.k8s.io/v1',
    kind: 'RoleList',
    items: [
      {
        apiVersion: 'rbac.authorization.k8s.io/v1',
        kind: 'Role',
        metadata: {
          labels: conprof.config.commonLabels,
          name: conprof.config.name,
          namespace: conprof.config.namespace,
        },
        rules: [
          {
            apiGroups: [
              '',
            ],
            resources: [
              'services',
              'endpoints',
              'pods',
            ],
            verbs: [
              'get',
              'list',
              'watch',
            ],
          },
        ],
      }
      for namespace in conprof.config.namespaces
    ],
  },

  secret: {
    apiVersion: 'v1',
    kind: 'Secret',
    metadata: {
      labels: conprof.config.commonLabels,
      name: conprof.config.name,
      namespace: conprof.config.namespace,
    },
    stringData: {
      'conprof.yaml': std.manifestYamlDoc(conprof.config.rawconfig),
    },
  },

  objectStorageSecret: if conprof.config.bucketConfig == null then {} else {
    apiVersion: 'v1',
    kind: 'Secret',
    metadata: {
      labels: conprof.config.commonLabels,
      name: conprof.config.name + '-objectstorage',
      namespace: conprof.config.namespace,
    },
    stringData: {
      'conprof.yaml': std.manifestYamlDoc({
        type: 's3',
        config: {
          bucket: conprof.config.bucketConfig.bucketName,
          endpoint: conprof.config.bucketConfig.endpoint,
          insecure: conprof.config.bucketConfig.insecure,
          access_key: conprof.config.bucketConfig.accessKey,
          secret_key: conprof.config.bucketConfig.secretKey,
        },
      }),
    },
  },

  statefulset: {
    apiVersion: 'apps/v1',
    kind: 'StatefulSet',
    metadata: {
      labels: conprof.config.commonLabels,
      name: conprof.config.name,
      namespace: conprof.config.namespace,
    },
    spec: {
      podManagementPolicy: 'Parallel',
      selector: {
        matchLabels: conprof.config.podLabelSelector,
      },
      serviceName: conprof.service.metadata.name,
      template: {
        metadata: {
          labels: conprof.config.commonLabels,
        },
        spec: {
          containers: [
            {
              args: [
                'all',
                '--storage.tsdb.path=/conprof',
                '--config.file=/etc/conprof/conprof.yaml',
              ] + if conprof.config.symbolServerURL == '' then [] else [
                '--symbol-server-url=' + conprof.config.symbolServerURL,
              ] + if conprof.config.bucketConfig == null then [] else [
                '--objstore.config=$(OBJSTORE_CONFIG)',
              ],
              image: conprof.config.image,
              name: 'conprof',
              env: if conprof.config.bucketConfig == null then [] else [{
                name: 'OBJSTORE_CONFIG',
                valueFrom: {
                  secretKeyRef: {
                    key: 'conprof.yaml',
                    name: conprof.objectStorageSecret.metadata.name,
                  },
                },
              }],
              ports: [
                {
                  containerPort: 10902,
                  name: 'http',
                },
                {
                  containerPort: 10901,
                  name: 'grpc',
                },
              ],
              volumeMounts: [
                {
                  mountPath: '/conprof',
                  name: 'storage',
                  readOnly: false,
                },
                {
                  mountPath: '/etc/conprof',
                  name: 'config',
                  readOnly: false,
                },
              ],
            },
          ],
          nodeSelector: {
            'kubernetes.io/os': 'linux',
          },
          serviceAccountName: conprof.serviceAccount.metadata.name,
          volumes: [
            {
              emptyDir: {},
              name: 'storage',
            },
            {
              name: 'config',
              secret: {
                secretName: conprof.secret.metadata.name,
              },
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
      labels: conprof.config.commonLabels,
      name: conprof.config.name,
      namespace: conprof.config.namespace,
    },
  },

  service: {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: {
      labels: conprof.config.commonLabels,
      name: conprof.config.name,
      namespace: conprof.config.namespace,
    },
    spec: {
      ports: [
        {
          name: 'http',
          port: 10902,
          targetPort: 'http',
        },
        {
          name: 'grpc',
          port: 10901,
          targetPort: 'grpc',
        },
      ],
      selector: conprof.config.podLabelSelector,
    },
  },

  withServiceMonitor:: {
    local conprof = self,
    serviceMonitor: {
      apiVersion: 'monitoring.coreos.com/v1',
      kind: 'ServiceMonitor',
      metadata: {
        name: 'conprof',
        namespace: conprof.config.namespace,
        labels: conprof.config.commonLabels,
      },
      spec: {
        selector: {
          matchLabels: conprof.config.podLabelSelector,
        },
        endpoints: [
          {
            port: 'http',
            interval: '30s',
          },
        ],
      },
    },
  },

  withConfigMap:: {
    local conprof = self,

    configmap: {
      apiVersion: 'v1',
      kind: 'ConfigMap',
      metadata: {
        name: 'conprof',
        namespace: conprof.config.namespace,
        labels: conprof.config.commonLabels,
      },
      data: {
        'conprof.yaml': std.manifestYamlDoc(conprof.config.rawconfig),
      },
    },

    statefulset+: {
      spec+: {
        template+: {
          spec+: {
            volumes:
              std.map(
                function(v) if v.name == 'config' then v {
                  secret:: null,
                  configMap: {
                    name: conprof.configmap.metadata.name,
                  },
                } else v,
                super.volumes
              ),
          },
        },
      },
    },

    secret:: null,
  },
}
