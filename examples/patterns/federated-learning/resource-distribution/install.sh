#!/bin/bash

set -eu
set -o pipefail
trap "trap - SIGTERM && kill -- -$$" SIGINT SIGTERM EXIT

export NAMESPACE=resource-distribution
export SCENARIO=$(dirname -- "$0")/manifest.yml
export REPORTS=${HOME}/frisbee-reports/${NAMESPACE}/
export DEPENDENCIES=(./charts/system/ ./examples/apps/fedbed/)

# Prepare the Reporting folder
mkdir -p "${REPORTS}"

# Copy the manifest
cp "${SCENARIO}" "${REPORTS}"

# Submit the scenario and follow server logs
kubectl-frisbee submit test "${NAMESPACE}" "${SCENARIO}" "${DEPENDENCIES[@]}" --logs server |& tee "${REPORTS}"/logs &

# Give a headstart
sleep 10

# Report the scenario
kubectl-frisbee report test "${NAMESPACE}" "${REPORTS}" --pdf --data --aggregated-pdf --wait
