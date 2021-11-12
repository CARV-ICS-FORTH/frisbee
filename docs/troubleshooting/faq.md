## This is a WIP FAQ for Frisbee

## FAQ

**Question:** What is Frisbee ?

**Answer:** Frisbee is a test-suite for Kubernetes.

##

**Q:** The service seems fine, but I get a Failed message.

**A:** The service run in a Pod that may host multiple containers. The application contrainer, the telemetry container,
and so on. Given that, if the application seems fine, it is perhaps one of the sidecar containers that has failed.

##

**Q:  I changed some templates, but the changes does not seem to affect the workflow.**

**A:** The changes are local and must be posted to the Kubernetes API. To update all templates use:

`find examples/templates -name "*.yml" -exec kubectl apply -f {} \;`

##

**Q: My experiment was running perfectly a few hours ago. Now, nothing works.**

**A:** A possible explanation is that you do not specify the container version. If so, the latest version is retrieved
for each run. And if there are incompatibilities between version, these incompatibilities will be reflected to your
experiment.

## Pod remains in ContainerCreating status with "error processing PVC ..: PVC is not bound"

See
https://github.com/openebs/openebs/issues/2915
https://github.com/kubernetes/kubernetes/issues/89953

Note: this is supposedly fixed by changing the domain abstraction from NodeName to NodeSelector

###                                              

##