# install frisbee via helm

helm install --generate-name --debug --create-namespace -n frisbee-testing ./
--kubeconfig=/home/fnikol/.kube/config.local

# copy from remote cluster to local file

kubectl --kubeconfig=/home/fnikol/.kube/config.evolve cp karvdash-fnikol/fio-jedi5:/dev/shm/pipe ~
/Workspace/projects/experiments/frisbee/fio_container.txt