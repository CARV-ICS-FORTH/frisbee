#!/bin/bash

export NAMESPACE=ml-backend
export SCENARIO=$(dirname -- "$0")/manifest.yml
export REPORTS=${HOME}/frisbee-reports/${NAMESPACE}/
export DEPENDENCIES=(./charts/system/ ./charts/federated-learning/fedbed/)

# Prepare the Reporting folder
mkdir -p "${REPORTS}"

# Submit the scenario and follow logs
kubectl-frisbee submit test "${NAMESPACE}" "${SCENARIO}" "${DEPENDENCIES[@]}"

# Copy the manifest
cp "${SCENARIO}" "${REPORTS}"

# wait for the scenario to be submitted
sleep 10

# Report the scenario
kubectl-frisbee report test "${NAMESPACE}" "${REPORTS}" --pdf --data --aggregated-pdf --wait
