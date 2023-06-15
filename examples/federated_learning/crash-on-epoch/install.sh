#!/bin/bash

set -eu
set -o pipefail

export NAMESPACE=crash-on-epoch
export SCENARIO=$(dirname -- "$0")/manifest.yml
export REPORTS=${HOME}/frisbee-reports/${NAMESPACE}/
export DEPENDENCIES=(./charts/system/ ./charts/federated-learning/fedbed/)

# Prepare the Reporting folder
mkdir -p "${REPORTS}"

# Copy the manifest
cp "${SCENARIO}" "${REPORTS}"

# Submit the scenario and follow logs
kubectl-frisbee submit test "${NAMESPACE}" "${SCENARIO}" "${DEPENDENCIES[@]}"

# wait for the scenario to be submitted
sleep 10

# Report the scenario
kubectl-frisbee report test "${NAMESPACE}" "${REPORTS}" --pdf --data --aggregated-pdf --wait
