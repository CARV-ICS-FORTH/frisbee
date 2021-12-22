## This is a WIP FAQ for Frisbee

## FAQ

**Question:** What is Frisbee ?

**Answer:** Frisbee is a test-suite for Kubernetes.

##

**Q:** **The service seems fine, but I get a Failed message.**

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

##

**Q: All I see in Grafana is a dot. There are no lines**

**A:** This is likely to happen when the duration of the experiment is too short. In general, we use 1/4 resolution in
order to make Grafana plots more readable. If you wish for a greater granularity, you can edit the chart and change
resolution to 1/1.

##

**Q: My plots in Grafana are not in line. The times are different**

**A:** This is likely to happen if you have a change the resolution of one graph, without changing the other. Go and set
the same resolution everywhere.

##