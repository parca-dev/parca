local k3 = import 'ksonnet/ksonnet.beta.3/k.libsonnet';
local k = import 'ksonnet/ksonnet.beta.4/k.libsonnet';

{
  local conprof = self,

  name:: error 'must provide name',
  namespace:: error 'must provide namespace',
  image:: error 'must provide image',
  version:: error 'must set version',

  namespaces:: [conprof.namespace],

  commonLabels:: {
    'app.kubernetes.io/name': 'conprof',
    'app.kubernetes.io/version': conprof.version,
  },

  podLabels:: {
    [labelName]: conprof.commonLabels[labelName]
    for labelName in std.objectFields(conprof.commonLabels)
    if !std.setMember(labelName, ['app.kubernetes.io/version'])
  },

  rawconfig:: {
    scrape_configs: [{
      job_name: 'conprof',
      kubernetes_sd_configs: [{
        namespaces: { names: conprof.namespaces },
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

  roleBindings:
    local roleBinding = k.rbac.v1.roleBinding;

    local newSpecificRoleBinding(namespace) =
      roleBinding.new() +
      roleBinding.mixin.metadata.withName(conprof.name) +
      roleBinding.mixin.metadata.withNamespace(namespace) +
      roleBinding.mixin.metadata.withLabels(conprof.commonLabels) +
      roleBinding.mixin.roleRef.withApiGroup('rbac.authorization.k8s.io') +
      roleBinding.mixin.roleRef.withName(conprof.name) +
      roleBinding.mixin.roleRef.mixinInstance({ kind: 'Role' }) +
      roleBinding.withSubjects([{ kind: 'ServiceAccount', name: conprof.name, namespace: conprof.namespace }]);

    local roleBindingList = k3.rbac.v1.roleBindingList;
    roleBindingList.new([newSpecificRoleBinding(x) for x in conprof.namespaces]),
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
      role.mixin.metadata.withName(conprof.name) +
      role.mixin.metadata.withNamespace(namespace) +
      role.mixin.metadata.withLabels(conprof.commonLabels) +
      role.withRules(coreRule);

    local roleList = k3.rbac.v1.roleList;
    roleList.new([newSpecificRole(x) for x in conprof.namespaces]),
  configsecret:
    local secret = k.core.v1.secret;
    secret.new('conprof-config', { 'conprof.yaml': std.base64(std.manifestYamlDoc(conprof.rawconfig)) }) +
    secret.mixin.metadata.withNamespace(conprof.namespace) +
    secret.mixin.metadata.withLabels(conprof.commonLabels),
  statefulset:
    local statefulset = k.apps.v1.statefulSet;
    local container = statefulset.mixin.spec.template.spec.containersType;
    local volume = statefulset.mixin.spec.template.spec.volumesType;
    local containerPort = container.portsType;
    local containerVolumeMount = container.volumeMountsType;
    local podSelector = statefulset.mixin.spec.template.spec.selectorType;

    local c = [
      container.new('conprof', conprof.image) +
      container.withArgs([
        'all',
        '--storage.tsdb.path=/conprof',
        '--config.file=/etc/conprof/conprof.yaml',
      ]) +
      container.withPorts([{ name: 'http', containerPort: 8080 }]) +
      container.withVolumeMounts([
        containerVolumeMount.new('storage', '/conprof'),
        containerVolumeMount.new('config', '/etc/conprof'),
      ],),
    ];

    { apiVersion: 'apps/v1', kind: 'StatefulSet' } +
    statefulset.mixin.metadata.withName(conprof.name) +
    statefulset.mixin.metadata.withNamespace(conprof.namespace) +
    statefulset.mixin.metadata.withLabels(conprof.commonLabels) +
    statefulset.mixin.spec.withPodManagementPolicy('Parallel') +
    statefulset.mixin.spec.withServiceName(conprof.name + '-governing-service') +
    statefulset.mixin.spec.selector.withMatchLabels(conprof.podLabels) +
    statefulset.mixin.spec.template.metadata.withLabels(conprof.podLabels) +
    statefulset.mixin.spec.template.spec.withContainers(c) +
    statefulset.mixin.spec.template.spec.withNodeSelector({ 'kubernetes.io/os': 'linux' }) +
    statefulset.mixin.spec.template.spec.withVolumes([
      volume.fromEmptyDir('storage'),
      volume.fromSecret('config', 'conprof-config'),
    ]) +
    statefulset.mixin.spec.template.spec.withServiceAccountName(conprof.name),

  serviceAccount:
    local serviceAccount = k.core.v1.serviceAccount;

    serviceAccount.new(conprof.name) +
    serviceAccount.mixin.metadata.withNamespace(conprof.namespace) +
    serviceAccount.mixin.metadata.withLabels(conprof.commonLabels),

  service:
    local service = k.core.v1.service;
    local servicePort = service.mixin.spec.portsType;

    local httpPort = servicePort.newNamed('http', 8080, 'http');

    service.new(conprof.name + '-governing-service', conprof.statefulset.spec.selector.matchLabels, [httpPort]) +
    service.mixin.metadata.withNamespace(conprof.namespace) +
    service.mixin.metadata.withLabels(conprof.commonLabels) +
    service.mixin.spec.withClusterIp('None'),

  mixin:: {
    serviceMonitor:
      {
        apiVersion: 'monitoring.coreos.com/v1',
        kind: 'ServiceMonitor',
        metadata: {
          name: 'conprof',
          namespace: conprof.namespace,
          labels: conprof.commonLabels,
        },
        spec: {
          jobLabel: 'app',
          selector: {
            matchLabels: {
              app: 'conprof',
            },
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
}
