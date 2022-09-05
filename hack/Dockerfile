FROM ubuntu

RUN apt-get update && apt-get install -y microk8s

RUN snap version

#RUN snap install snapcraft --classic

# Install Microk8s
RUN snap install microk8s --classic --channel=stable

# Enable addons
#RUN microk8s enable dns ingress helm3 rbac

# Start microk8s
#RUN microk8s start