#!/usr/bin/env bash

set -e

cd /templates || exit 1

START_TIME=$(date +%s)
export START_TIME

# Generate all proposals from templates
# Loop through all proposal templates and generate the proposals
for template in *.template.json; do
  echo "Generating file from template: $template"
  envsubst < $template > /generated/${template%.template.json}.json
done

START_TIME=
