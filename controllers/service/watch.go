package service

/*
func (r *Reconciler) Watcher(ctx context.Context, obj *v1alpha1.Service) error {
	informer, err := r.Manager.GetCache().GetInformer(ctx, &corev1.Pod{})
	if err != nil {
		return errors.Wrapf(err, "unable to get informer")
	}

	informer.AddEventHandler(cachetypes.FilteringResourceEventHandler{
		FilterFunc: lifecycle.MergeFilters(
			lifecycle.FilterByParent(obj),
		),
		Handler: cachetypes.ResourceEventHandlerFuncs{
			AddFunc:    r.addPod(obj),
			UpdateFunc: r.updatePod(obj),
			DeleteFunc: r.deletePod(obj),
		},
	})

	return nil
}

// When a pod is created, enqueue the service that manages it and update its expectations.
func (r *Reconciler) addPod(owner *v1alpha1.Service) func(obj interface{}) {
	return func(obj interface{}) {
		pod := obj.(*corev1.Pod)

		logrus.Warn("Add pod ", pod.GetName())

		if pod.DeletionTimestamp != nil {
			// on a restart of the controller manager, it's possible a new pod shows up in a state that
			// is already pending deletion. Prevent the pod from being a creation observation.
			// r.deletePod(owner)(pod)
			return
		}
	}
}

// When a pod is updated, figure out what service manage it and wake them
// up. If the labels of the pod have changed we need to awaken both the old
// and new replica set. old and cur must be *v1.Pod types.
func (r *Reconciler) updatePod(owner *v1alpha1.Service) func(old, cur interface{}) {
	return func(old, cur interface{}) {
		curPod := cur.(*corev1.Pod)
		oldPod := old.(*corev1.Pod)

		if curPod.ResourceVersion == oldPod.ResourceVersion {
			// Periodic resync will send update events for all known pods.
			// Two different versions of the same pod will always have different RVs.
			return
		}

		if curPod.DeletionTimestamp != nil {
			// when a pod is deleted gracefully it's deletion timestamp is first modified to reflect a grace period,
			// and after such time has passed, the kubelet actually deletes it from the store. We receive an update
			// for modification of the deletion timestamp and expect an rs to create more replicas asap, not wait
			// until the kubelet actually deletes the pod. This is different from the Phase of a pod changing, because
			// an rs never initiates a phase change, and so is never asleep waiting for the same.
			r.deletePod(curPod)
			if labelChanged {
				// we don't need to check the oldPod.DeletionTimestamp because DeletionTimestamp cannot be unset.
				rsc.deletePod(oldPod)
			}
			return
		}

		logrus.Warn("Update pod ", pod.GetName())
		logrus.Warn("Status ")
	}
}

// When a pod is deleted, enqueue the service that manages the pod and update its expectations.
// obj could be an *v1.Pod, or a DeletionFinalStateUnknown marker item.
func (r *Reconciler) deletePod(owner *v1alpha1.Service) func(obj interface{}) {
	return func(obj interface{}) {
		pod := obj.(*corev1.Pod)

		logrus.Warn("delete pod ", pod.GetName())
	}
}


*/
