#!/bin/bash
set -eum

echo "Building the container with snapd"
docker build . -t snapd_base

echo "Run the snapd container as daemon"
execution=$(docker  run -d --privileged -v /sys/fs/cgroup:/sys/fs/cgroup --entrypoint /lib/systemd/systemd  snapd_base)

echo "Setup exit trap to stop the container ${execution}"
# trap "docker stop ${execution}" EXIT
trap "docker exec -ti ${execution} bash" EXIT

echo "Connecting to snapd ..."
docker  exec "${execution}"  bash -c  "while [ ! -e /run/snapd.socket ]; do  sleep 2; done"

sleep 5

echo "Installing microk8s to the container"
docker  exec -t "${execution}"  snap install microk8s --classic

echo "Building aliases"
docker exec -t "${execution}" snap alias microk8s.kubectl kubectl
docker exec -t "${execution}" snap alias microk8s.helm3 helm

echo "Fixing configurations"
docker exec -t "${execution}" microk8s config > ~/.kube/config

echo "Starting microk8s"
docker  exec -t "${execution}"  microk8s start

echo "Waiting for microk8s to become ready"
docker  exec -t "${execution}"  microk8s status --wait-ready

echo "Installing microk8s addons to the container"
docker  exec -t "${execution}" microk8s enable dns ingress helm3 rbac


echo "Installing Frisbee to the container"
docker  exec -t "${execution}"  bash -c "curl -sSLf https://frisbee.dev/install.sh | bash"
docker  exec -t "${execution}"  kubectl frisbee install production

echo "Update tag for the updated image"
docker tag "${execution}" microk8s-frisbee

echo "Commit the updated image"
docker commit snapd_base microk8s-frisbee

echo "Stop the building container"
docker stop "${execution}"

#echo "Remove base image"
#docker stop snapd_base