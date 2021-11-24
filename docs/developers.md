#### Lint Charts

```bash
yamllint ./platform/Chart.yaml
```

```
docker run quay.io/helmpack/chart-testing:lates ct lint --target-branch=main --check-version-increment=false
```
