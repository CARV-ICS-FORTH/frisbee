# Grafana

## Create PDF from dashboards

To fetch a PDF from Grafana follow the instructions of: https://gist.github.com/svet-b/1ad0656cd3ce0e1a633e16eb20f66425

Briefly,

1. Install Node.js on your local workstation
2. wget https://gist.githubusercontent.com/svet-b/1ad0656cd3ce0e1a633e16eb20f66425/raw/grafana_pdf.js
3. execute the grafana_fs.js over node.ns

> node grafana_pdf.js "http://grafana.localhost/d/A2EjFbsMk/ycsb-services?viewPanel=74" "":"" output.pdf

######                                                          

###### Permissions

By default, Grafana is configured without any login requirements, so we must leave this field blank

"":"" denotes empty username:password.

