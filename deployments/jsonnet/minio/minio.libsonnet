// These are the defaults for this components configuration.
// When calling the function to generate the component's manifest,
// you can pass an object structured like the default to overwrite default values.
local defaults = {
  local defaults = self,
  namespace: error 'must provide namespace',
  accessKey: error 'must provide accessKey',
  secretKey: error 'must provide secretKey',
  bucketName: error 'must provide bucketName',

  commonLabels:: { 'app.kubernetes.io/name': 'minio' },
};

function(params) {
  local minio = self,

  // Combine the defaults and the passed params to make the component's config.
  config:: defaults + params,

  deployment: {
    apiVersion: 'apps/v1',
    kind: 'Deployment',
    metadata: {
      name: 'minio',
      namespace: minio.config.namespace,
    },
    spec: {
      selector: {
        matchLabels: minio.config.commonLabels,
      },
      strategy: { type: 'Recreate' },
      template: {
        metadata: {
          labels: minio.config.commonLabels,
        },
        spec: {
          containers: [
            {
              command: [
                '/bin/sh',
                '-c',
                |||
                  mkdir -p /storage/%s && \
                  /usr/bin/minio server /storage
                ||| % minio.config.bucketName,
              ],
              env: [
                {
                  name: 'MINIO_ACCESS_KEY',
                  value: minio.config.accessKey,
                },
                {
                  name: 'MINIO_SECRET_KEY',
                  value: minio.config.secretKey,
                },
              ],
              image: 'minio/minio',
              name: 'minio',
              ports: [
                { containerPort: 9000 },
              ],
              volumeMounts: [
                { mountPath: '/storage', name: 'storage' },
              ],
            },
          ],
          volumes: [{
            name: 'storage',
            persistentVolumeClaim: { claimName: 'minio' },
          }],
        },
      },
    },
  },

  pvc: {
    apiVersion: 'v1',
    kind: 'PersistentVolumeClaim',
    metadata: {
      labels: minio.config.commonLabels,
      name: 'minio',
      namespace: minio.config.namespace,
    },
    spec: {
      accessModes: ['ReadWriteOnce'],
      resources: {
        requests: { storage: '10Gi' },
      },
    },
  },

  service: {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: {
      name: 'minio',
      namespace: minio.config.namespace,
    },
    spec: {
      ports: [
        { port: 9000, protocol: 'TCP', targetPort: 9000 },
      ],
      selector: minio.config.commonLabels,
      type: 'ClusterIP',
    },
  },
}
