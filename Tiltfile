## Parca

docker_build('quay.io/parca/parca:dev', '.',
    dockerfile='Dockerfile.dev',
    only=['./cmd', './pkg', './internal', './proto', './ui', './go.mod', './go.sum', 'parca.yaml'],
)
k8s_yaml('deploy/manifests/parca-deployment.yaml')
k8s_resource('parca', port_forwards=7070)

## Parca UI
## It's redundant. Parca already serves the UI, just in case if someone wants to iterate soley on UI.
docker_build('quay.io/parca-dev/parca-ui:dev', './ui',
    entrypoint='yarn workspace @parca/web dev',
    dockerfile='./ui/Dockerfile.dev',
    live_update=[
        sync('./ui', '/app'),
        run('cd /app && yarn install', trigger=['./package.json', './yarn.lock']),
    ],
)
k8s_yaml('deploy/manifests/parca-ui-dev-deployment.yaml')
k8s_resource('parca-ui', port_forwards=3000)

## Parca Agent

# Until Parca will be public we need to supply a personal access token for the builds.
docker_build('quay.io/parca/parca-agent', './tmp/parca-agent', build_args={'TOKEN': read_file('./tmp/personal_access_token')})
k8s_yaml('deploy/manifests/parca-agent-daemonSet.yaml')
