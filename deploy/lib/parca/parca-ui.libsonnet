// These are the defaults for this components configuration.
// When calling the function to generate the component's manifest,
// you can pass an object structured like the default to overwrite default values.
local defaults = {
  local defaults = self,

  name: 'parca-ui',
  namespace: error 'must provide namespace',
  version: error 'must provide version',
  image: error 'must provide image',
  replicas: error 'must provide replicas',
  apiEndpoint: '',

  commonLabels:: {
    'app.kubernetes.io/name': 'parca-ui',
    'app.kubernetes.io/instance': defaults.name,
    'app.kubernetes.io/version': defaults.version,
    'app.kubernetes.io/component': 'ui',
  },

  podLabelSelector:: {
    [labelName]: defaults.commonLabels[labelName]
    for labelName in std.objectFields(defaults.commonLabels)
    if !std.setMember(labelName, ['app.kubernetes.io/version'])
  },
};

function(params) {
  local ui = self,

  // Combine the defaults and the passed params to make the component's config.
  config:: defaults + params,

  deployment: {
    apiVersion: 'apps/v1',
    kind: 'Deployment',
    metadata: {
      name: ui.config.name,
      namespace: ui.config.namespace,
      labels: ui.config.commonLabels,
    },
    spec: {
      replicas: ui.config.replicas,
      selector: {
        matchLabels: ui.config.podLabelSelector,
      },
      template: {
        metadata: {
          labels: ui.config.commonLabels,
        },
        spec: {
          containers: [
            {
              image: ui.config.image,
              name: 'parca-ui',
              ports: [
                {
                  containerPort: 3000,
                  name: 'http',
                },
              ],
              env:
                (if ui.config.apiEndpoint != '' then [{
                   name: 'NEXT_PUBLIC_API_ENDPOINT',
                   value: ui.config.apiEndpoint,
                 }] else []),
            },
          ],
          serviceAccountName: ui.serviceAccount.metadata.name,
        },
      },
    },
  },

  service: {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: {
      name: ui.config.name,
      namespace: ui.config.namespace,
      labels: ui.config.commonLabels,
    },
    spec: {
      ports: [
        {
          name: 'http',
          port: 3000,
          protocol: 'TCP',
          targetPort: 3000,
        },
      ],
      selector: ui.config.podLabelSelector,
    },
  },

  serviceAccount: {
    apiVersion: 'v1',
    kind: 'ServiceAccount',
    metadata: {
      name: ui.config.name,
      namespace: ui.config.namespace,
      labels: ui.config.commonLabels,
    },
  },
}
