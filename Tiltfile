docker_prune_settings(num_builds=5)

# Parca

## API Only
docker_build('quay.io/parca/parca:dev', '.',
    dockerfile='Dockerfile.go.dev',
    only=['./cmd', './pkg', './internal', './proto', './gen', './go.mod', './go.sum', 'parca.yaml'],
)

## All-in-one
# docker_build('quay.io/parca/parca:dev', '.',
#     dockerfile='Dockerfile.dev',
#     only=['./cmd', './pkg', './internal', './proto', './gen', './ui', './go.mod', './go.sum', 'parca.yaml'],
# )

k8s_yaml('deploy/manifests/parca-deployment.yaml')
k8s_resource('parca', port_forwards=[7070, 40000])

## UI
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

docker_build('quay.io/parca/parca-agent:dev', './tmp/parca-agent',
    dockerfile='./tmp/parca-agent/Dockerfile.dev',
    # Until Parca will be public we need to supply a personal access token for the builds.
    build_args={'TOKEN': read_file('./tmp/personal_access_token')},
)
k8s_yaml('deploy/manifests/parca-agent-daemonSet.yaml')
k8s_resource('parca-agent', port_forwards=[7071])
