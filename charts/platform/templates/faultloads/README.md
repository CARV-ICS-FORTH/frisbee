Faultload is a collection of faults that may happen to the specific Kubernets components.


Reserved Labels:
  # To gather all information regarding faultloads
  app.frisbee.io/component: faultload

Reserved Annotations:
  faultload.frisbee.io/apply-to: the CRD to which the faultload is applicable

  Example:     faultload.frisbee.io/apply-to: core/v1/volumes

