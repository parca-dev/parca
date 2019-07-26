local k3 = import 'ksonnet/ksonnet.beta.3/k.libsonnet';
local k = import 'ksonnet/ksonnet.beta.4/k.libsonnet';

{
  conprof+:: {
    config+:: {
      name: 'conprof-main',
      namespace: 'conprof',
      image: 'quay.io/conprof/conprof:v0.1.0-dev',

      namespaces: [$.conprof.config.namespace],

      rawconfig:
        {
          scrape_configs: [{
            job_name: 'conprof',
            kubernetes_sd_configs: [{
              namespaces: { names: $.conprof.config.namespaces },
              role: 'pod',
            }],
            relabel_configs: [
              {
                action: 'keep',
                regex: 'conprof.*',
                source_labels: ['__meta_kubernetes_pod_name'],
              },
            ],
            scrape_interval: '1m',
            scrape_timeout: '1m',
          }],
        },
    },

    roleBindings:
      local roleBinding = k.rbac.v1.roleBinding;

      local newSpecificRoleBinding(namespace) =
        roleBinding.new() +
        roleBinding.mixin.metadata.withName($.conprof.config.name) +
        roleBinding.mixin.metadata.withNamespace(namespace) +
        roleBinding.mixin.roleRef.withApiGroup('rbac.authorization.k8s.io') +
        roleBinding.mixin.roleRef.withName($.conprof.config.name) +
        roleBinding.mixin.roleRef.mixinInstance({ kind: 'Role' }) +
        roleBinding.withSubjects([{ kind: 'ServiceAccount', name: $.conprof.config.name, namespace: $.conprof.config.namespace }]);

      local roleBindingList = k3.rbac.v1.roleBindingList;
      roleBindingList.new([newSpecificRoleBinding(x) for x in $.conprof.config.namespaces]),
    roles:
      local role = k.rbac.v1.role;
      local policyRule = role.rulesType;
      local coreRule = policyRule.new() +
                       policyRule.withApiGroups(['']) +
                       policyRule.withResources([
                         'services',
                         'endpoints',
                         'pods',
                       ]) +
                       policyRule.withVerbs(['get', 'list', 'watch']);

      local newSpecificRole(namespace) =
        role.new() +
        role.mixin.metadata.withName($.conprof.config.name) +
        role.mixin.metadata.withNamespace(namespace) +
        role.withRules(coreRule);

      local roleList = k3.rbac.v1.roleList;
      roleList.new([newSpecificRole(x) for x in $.conprof.config.namespaces]),
    configsecret:
      local secret = k.core.v1.secret;
      secret.new('conprof-config', { 'conprof.yaml': std.base64(std.manifestYamlDoc($.conprof.config.rawconfig)) }) +
      secret.mixin.metadata.withNamespace($.conprof.config.namespace),
    statefulset:
      local statefulset = k.apps.v1.statefulSet;
      local container = statefulset.mixin.spec.template.spec.containersType;
      local volume = statefulset.mixin.spec.template.spec.volumesType;
      local containerPort = container.portsType;
      local containerVolumeMount = container.volumeMountsType;
      local podSelector = statefulset.mixin.spec.template.spec.selectorType;
      local podLabels = { app: 'conprof' };

      local conprof =
        container.new('conprof', $.conprof.config.image) +
        container.withArgs([
          'all',
          '--storage.tsdb.path=/conprof',
          '--config.file=/etc/conprof/conprof.yaml',
        ]) +
        container.withPorts([{ containerPort: 8080 }]) +
        container.withVolumeMounts([
          containerVolumeMount.new('storage', '/conprof'),
          containerVolumeMount.new('config', '/etc/conprof'),
        ],);

      local c = [conprof];

      { apiVersion: 'apps/v1', kind: 'StatefulSet' } +
      statefulset.mixin.metadata.withName($.conprof.config.name) +
      statefulset.mixin.metadata.withNamespace($.conprof.config.namespace) +
      statefulset.mixin.metadata.withLabels(podLabels) +
      statefulset.mixin.spec.withPodManagementPolicy('Parallel') +
      statefulset.mixin.spec.withServiceName($.conprof.config.name + '-governing-service') +
      statefulset.mixin.spec.selector.withMatchLabels(podLabels) +
      statefulset.mixin.spec.template.metadata.withLabels(podLabels) +
      statefulset.mixin.spec.template.spec.withContainers(c) +
      statefulset.mixin.spec.template.spec.withNodeSelector({ 'kubernetes.io/os': 'linux' }) +
      statefulset.mixin.spec.template.spec.withVolumes([
        volume.fromEmptyDir('storage'),
        volume.fromSecret('config', 'conprof-config'),
      ]) +
      statefulset.mixin.spec.template.spec.withServiceAccountName($.conprof.config.name),

    serviceAccount:
      local serviceAccount = k.core.v1.serviceAccount;

      serviceAccount.new($.conprof.config.name) +
      serviceAccount.mixin.metadata.withNamespace($.conprof.config.namespace),

    service:
      local service = k.core.v1.service;
      local servicePort = service.mixin.spec.portsType;

      local httpPort = servicePort.newNamed('http', 8080, 'http');

      service.new($.conprof.config.name + '-governing-service', $.conprof.statefulset.spec.selector.matchLabels, [httpPort]) +
      service.mixin.metadata.withNamespace($.conprof.config.namespace) +
      service.mixin.metadata.withLabels({ app: 'conprof' }) +
      service.mixin.spec.withClusterIp('None'),

    //serviceMonitor:
    //  {
    //    apiVersion: 'monitoring.coreos.com/v1',
    //    kind: 'ServiceMonitor',
    //    metadata: {
    //      name: 'conprof',
    //      namespace: $.conprof.config.namespace,
    //      labels: {
    //        app: 'conprof',
    //      },
    //    },
    //    spec: {
    //      jobLabel: 'app',
    //      selector: {
    //        matchLabels: {
    //          app: 'conprof',
    //        },
    //      },
    //      endpoints: [
    //        {
    //          port: 'http',
    //          interval: '30s',
    //        },
    //      ],
    //    },
    //  },
  },
}
