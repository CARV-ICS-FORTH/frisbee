#!/bin/bash

set -eu
set -o pipefail
trap "trap - SIGTERM && kill -- -$$" SIGINT SIGTERM EXIT

export NAMESPACE=network-partition
export SCENARIO=$(dirname -- "$0")/manifest.yml
export REPORTS=${HOME}/frisbee-reports/${NAMESPACE}/
export DEPENDENCIES=(./charts/system/ ./charts/databases/cockroachdb ./charts/databases/ycsb)

# Prepare the Reporting folder
mkdir -p "${REPORTS}"

# Copy the manifest
cp "${SCENARIO}" "${REPORTS}"

# Submit the scenario and follow logs
kubectl-frisbee submit test "${NAMESPACE}" "${SCENARIO}" "${DEPENDENCIES[@]}"

# Give a headstart
sleep 10

# Report the scenario
kubectl-frisbee report test "${NAMESPACE}" "${REPORTS}" --pdf --data --aggregated-pdf --wait