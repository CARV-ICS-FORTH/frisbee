#!/bin/bash

export NAMESPACE=ml-backend
export SCENARIO=$(dirname -- "$0")/manifest.yml
export DEPENDENCIES=(./charts/system/ ./charts/federated-learning/fedbed/)

kubectl-frisbee submit test "${NAMESPACE}" "${SCENARIO}" "${DEPENDENCIES[@]}"
