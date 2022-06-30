docker_prune_settings(num_builds=5)

# Parca

## All-in-one
docker_build(
    'localhost:5000/parca:dev', '',
    dockerfile='Dockerfile.dev',
    only=[
        './cmd',
        './pkg',
        './proto',
        './gen',
        './ui',
        './go.mod',
        './go.sum',
        'parca.yaml',
    ],
)

k8s_yaml('deploy/tilt/parca-server-deployment.yaml')
k8s_resource('parca', port_forwards=[7070])

## API Only
# docker_build(
#     'localhost:5000.parca:dev', '.',
#     dockerfile='Dockerfile.go.dev',
#     only=['./cmd', './pkg', './proto', './gen', './go.mod', './go.sum', 'parca.yaml'],
# )

# TODO(kakkoyun): Make conditional.
## UI Only
# docker_build(
#     'localhost:5000/parca-ui:dev', './ui',
#     entrypoint='yarn workspace @parca/web dev',
#     dockerfile='./ui/Dockerfile.dev',
#     live_update=[
#         sync('./ui', '/app'),
#         run('cd /app && yarn install', trigger=['./package.json', './yarn.lock']),
#     ],
# )
# k8s_yaml('deploy/tilt/parca-ui-deployment.yaml')
# k8s_resource('parca-ui', port_forwards=3000)
