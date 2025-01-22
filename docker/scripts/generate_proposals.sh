#!/usr/bin/env bash

set -e

cd /templates || exit 1

# Generate all proposals from templates
# Loop through all proposal templates and generate the proposals
for template in *.template.json; do
  echo "Generating proposal from template: $template"
  envsubst < $template > /proposals/${template%.template.json}.json
done
