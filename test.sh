#!/bin/bash

declare -A TEST_ARRAY=(
  ["key1"]="value1"
  ["key2"]="value2"
)

for key in "${!TEST_ARRAY[@]}"; do
  echo "Key: $key, Value: ${TEST_ARRAY[$key]}"
done
