#!/bin/bash

yarn build
mkdir PATH_PREFIX_VAR
cp -r ./build/* ./PATH_PREFIX_VAR/
mv ./PATH_PREFIX_VAR ./build/
