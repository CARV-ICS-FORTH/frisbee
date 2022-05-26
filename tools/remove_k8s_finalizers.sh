set -em

# For more explanation check https://www.crybit.com/kubernetes-objects-deleting-stuck-in-terminating-state/

names=("$@")

echo "Expected format ./remove_k8s_finalizers.sh kind name0 name1 ..."

echo kubectl patch ${names[@]} --type json --patch=\'[ { "op": "remove", "path": "/metadata/finalizers" } ]\'