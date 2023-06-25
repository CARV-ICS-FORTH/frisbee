# The Frisbee Library

Popular applications, provided by Frisbee, ready to launch on Kubernetes
using [Kubernetes Helm](https://github.com/helm/helm).

## TL;DR

```bash
# Add Frisbee repo in Helm
$ helm repo add frisbee https://carv-ics-forth.github.io/frisbee/charts

# Install the Frisbee platform
$ helm install my-frisbee charts/frisbee

# Install the default system templates
$ helm install my-system charts/system

# Install the any of the available charts.
$ helm install my-example charts/iperf-2
```
