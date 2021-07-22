https://chaos-mesh.org/docs/development_guides/develop_a_new_chaos/

Make uninstall make install

// Will create new CRDS

Make update

Because of this bug, https://github.com/argoproj/argo-cd/issues/820, we cannot use the kubectl apply directly on the
CRDS. For this reason, we must uninstall and install all the crds at once.

make install KUBECONFIG="--kubeconfig /home/fnikol/.kube/config.evolve" NAMESPACE="-n karvdash-fnikol"

RESERVED LABELS:

* discover
* owner


* For mapping local directories to configMap
  https://itnext.io/helm-3-mapping-a-directory-of-files-into-a-container-ed6c54372df8