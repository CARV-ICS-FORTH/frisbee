== Operatator-SDK

=== Create a new Controller

* operator-sdk create api --group frisbee --version v1alpha1 --kind MyNewController --resource --controller
* operator-sdk create webhook --group frisbee --version v1alpha1 --kind MyNewController --defaulting
  --programmatic-validation

docker save frisbee:latest -o image.tar