#!/usr/bin/env bash
#
# Starts ConProf `all` mode on :8080 which collects profiles from itself and stores in ./data

trap 'kill 0' SIGTERM

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

WEB_ADDRESS="0.0.0.0:8080"
DATA_DIR="${DIR}/data"
CONPROF_EXECUTABLE=${CONPROF_EXECUTABLE:-"./conprof"}

if [ ! $(command -v "$CONPROF_EXECUTABLE") ]; then
  echo "Cannot find or execute ConProf binary ${CONPROF_EXECUTABLE}, you can override it by setting the CONPROF_EXECUTABLE env variable"
  exit 1
fi

${CONPROF_EXECUTABLE} all \
  --log.level=debug \
  --web.listen-address=${WEB_ADDRESS} \
  --storage.tsdb.path=${DATA_DIR} \
  --config.file="${DIR}/conprof.yaml" \
  --storage.tsdb.retention.time=15d

echo "conprof started; waiting for signal"

wait
