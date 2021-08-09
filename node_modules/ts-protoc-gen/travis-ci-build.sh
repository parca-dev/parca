#!/bin/bash
set -ex

npm run lint
npm test

npm run generate

git --no-pager diff

MODIFIED_FILES=$(git diff --name-only)
if [[ -n $MODIFIED_FILES ]]; then
  echo "ERROR: Changes detected in generated code, please run 'npm run generate' and check-in the changes."
  exit 1
fi
