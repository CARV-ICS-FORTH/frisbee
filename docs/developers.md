#### Lint Charts

```bash
yamllint ./platform/Chart.yaml
```

```
docker run quay.io/helmpack/chart-testing:lates ct lint --target-branch=main --check-version-increment=false
```

#### Working with MicroK8s’ built-in registry

```bash
# Install the registry
microk8s enable registry

# To upload images we have to tag them with localhost:32000/your-image before pushing them:
docker build . -t localhost:32000/mynginx:registry

# Now that the image is tagged correctly, it can be pushed to the registry:
docker push localhost:32000/mynginx
```

Pushing to this insecure registry may fail in some versions of Docker unless the daemon is explicitly configured to
trust this registry.

To address this we need to edit `/etc/docker/daemon.json` and add:

```json
{
  "insecure-registries": [
    "localhost:32000"
  ]
}
```

The new configuration should be loaded with a Docker daemon restart:

```bash
sudo systemctl restart docker
```

Source: https://microk8s.io/docs/registry-built-in
