#!/bin/bash

set -eux

export NAMESPACE=parallel-workflows
export SCENARIO=$(dirname -- "$0")/manifest.yml
export DEPENDENCIES=(./charts/system/ ./charts/federated-learning/fedbed/)
export REPORTS=${HOME}/frisbee-reports/${NAMESPACE}/

# Submit the scenario
kubectl-frisbee submit test "${NAMESPACE}" "${SCENARIO}" "${DEPENDENCIES[@]}"

# Copy the manifest
mkdir -p "${REPORTS}"
cp "${SCENARIO}" "${REPORTS}"

# Report the scenario
kubectl-frisbee report test "${NAMESPACE}" "${REPORTS}" --pdf --data --aggregated-pdf --wait

