https://chaos-mesh.org/docs/development_guides/develop_a_new_chaos/

Make uninstall make install

// Will create new CRDS

Make update

Because of this bug, https://github.com/argoproj/argo-cd/issues/820, we cannot use the kubectl apply directly on the
CRDS. For this reason, we must uninstall and install all the crds at once.