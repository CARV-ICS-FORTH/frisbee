# Virtual Slurm Cluster

The aim is to give an environment to test, introduce and practice the use and development over a Slurm Cluster (this is not an environment for production).


This is an adaptation of https://medium.com/analytics-vidhya/slurm-cluster-with-docker-9f242deee601

```
kubectl frisbee submit test demo ./examples/slurm-jupyter.yml ./ charts/system/
```

The ./examples/slurm-jupyter.yml is the scenario to run.
The `.` install dependencies to the templates of this chart/package.
The `../chart/system` install dependencies to the template of an external chart/package.


## Parameters