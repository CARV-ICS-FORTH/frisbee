<a href="https://www.vectorstock.com/royalty-free-vector/disc-golf-frisbee-eps-vector-25179185" target="_blank"><figure><img src="https://github.com/CARV-ICS-FORTH/frisbee/blob/main/docs/images/logo.jpg" width="400"></figure></a>


## Usage

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

  helm repo add <alias> https://<orgname>.github.io/helm-charts

If you had already added this repo earlier, run `helm repo update` to retrieve
the latest versions of the packages.  You can then run `helm search repo
<alias>` to see the charts.

To install the <chart-name> chart:

    helm install my-<chart-name> <alias>/<chart-name>

To uninstall the chart:

    helm delete my-<chart-name>
