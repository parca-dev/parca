#!/usr/bin/env bash

trap 'kill 0' SIGTERM

EXECUTABLE=${THANOS_EXECUTABLE:-"conprof"}

${EXECUTABLE} storage --log.level="debug" --http-address=":10902" --grpc-address=":10901" &
${EXECUTABLE} api     --log.level="debug" --http-address=":10912" --store="127.0.0.1:10901" &
${EXECUTABLE} sampler --log.level="debug" --http-address=":10922" --store="127.0.0.1:10901" --insecure --config.file="examples/conprof.yaml" &

echo "all started; waiting for signal"

wait

