#!/usr/bin/env bash
set -eo pipefail

echo "Generating Helios proto code"
cd proto
proto_dirs=$(find ./helios -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    if grep "option go_package" $file &> /dev/null ; then
      buf generate --template buf.gen.gogo.yml $file
    fi
  done
done

cd ..

# move proto files to the right places
cp -r helios-core/* ./
rm -rf helios-core

# Uncomment if needed
# ./scripts/protocgen-pulsar.sh