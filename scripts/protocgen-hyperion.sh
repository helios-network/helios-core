#!/usr/bin/env bash
set -eo pipefail

echo "Generating gogo proto code"
cd proto
proto_dirs=$(find ./helios/hyperion -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    if grep "option go_package" $file &> /dev/null ; then
      buf generate --template buf.gen.gogo.yml $file
    fi
  done
done

cd ..

DIRECTORY_TO_OVERLOAD="."

echo "Replacing strings in all files within $DIRECTORY_TO_OVERLOAD"

# Vérifier que le répertoire existe
if [[ ! -d "$DIRECTORY_TO_OVERLOAD" ]]; then
  echo "Error: Directory $DIRECTORY_TO_OVERLOAD does not exist."
  exit 1
fi

# Parcourir tous les fichiers du répertoire
for file in $(find "$DIRECTORY_TO_OVERLOAD" -type f); do
  echo "Processing file: $file"
  
  # Remplacer les chaînes dans chaque fichier
  awk '{
    gsub("github.com/gogo/protobuf/grpc", "github.com/cosmos/gogoproto/grpc");
    gsub("github.com/gogo/protobuf/proto", "github.com/cosmos/gogoproto/proto");
    gsub("github.com/gogo/protobuf/types", "github.com/cosmos/gogoproto/types");
    gsub("github_com_gogo_protobuf_types", "github_com_cosmos_gogoproto_types");
    print;
  }' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
done

echo "Replacement completed for all files in $DIRECTORY_TO_OVERLOAD."

# move proto files to the right places
cp -r helios/* ./
rm -rf helios

# ./scripts/protocgen-pulsar.sh