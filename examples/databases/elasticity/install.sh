#!/bin/bash

set -eux

export NAMESPACE=elasticity


export SCENARIO=$(dirname -- "$0")/manifest.yml
export REPORTS=${HOME}/frisbee-reports/${NAMESPACE}/
export DEPENDENCIES=(./charts/system/ ./charts/databases/cockroachdb ./charts/databases/ycsb)


# Submit the scenario
kubectl-frisbee submit test "${NAMESPACE}" "${SCENARIO}" "${DEPENDENCIES[@]}"

# Copy the manifest
mkdir -p "${REPORTS}"
cp "${SCENARIO}" "${REPORTS}"

# wait for the scenario to be submitted
sleep 10

# Report the scenario
kubectl-frisbee report test "${NAMESPACE}" "${REPORTS}" --pdf --data --aggregated-pdf --wait