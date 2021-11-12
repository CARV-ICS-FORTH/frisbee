Create a public Helm chart repository with GitHub Pages


What is Helm and a Helm chart repository

Helm, as per official claim, is “The package manager for Kubernetes”.

Indeed Helm really helps handling application deployments on Kubernetes, not only because it simplifies the application
release phase but also because Helm makes possible to easily manage variables in the Kubernetes manifest YAML files.

Once the charts are ready and you need to share them, the easiest way to do so is by uploading them to a chart 
repository. A Helm chart repository is where we host and share Helm packages and any HTTP server will do. Unluckily 
Helm does not include natively a tool for uploading charts to a remote chart repository server (Helm can serve the 
local repository via $ helm serve though).

We’ll take advantage of GitHub Pages for the purpose to share our charts.

What is GitHub Pages

GitHub Pages is a static site hosting service provided to you by GitHub, designed to host your pages directly from a 
GitHub repository. GitHub Pages is a perfect match for our purpose, since everything we need is to host a single 
index.yaml file along with a bunch of .tgz files.

Why not hosting all this stuff in your own web server? A managed service helps to reduce your operational overhead 
and risk (think to monitoring, patch management, security, backup services…) so you can focus on code and in what 
makes your business relevant for your customers.


Useful links:
https://medium.com/@mattiaperi/create-a-public-helm-chart-repository-with-github-pages-49b180dbb417
https://github.com/technosophos/tscharts