package service

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func (r *Reconciler) createKubeConfigMap(ctx context.Context, obj *v1alpha1.Service) ([]corev1.Volume, []corev1.VolumeMount, error) {
	/*
		if len(obj.Spec.ConfigPath) == 0 {
			return nil, nil, nil
		}

		// create configmap from local configuration
		// FIXME: https://github.com/kubernetes/kubernetes/pull/63362#issuecomment-386631005
		config := load.ConfigDir(obj.Spec.ConfigPath)

		configMap := corev1.ConfigMap{
			ObjectMeta: makeItOwned(obj),
			Immutable:  nil,
			Data: func() map[string]string {
				configFiles := make(map[string]string, len(config))
				for _, conf := range config {
					configFiles[conf.File] = string(conf.Payload)
				}

				return configFiles
			}(),
		}

		if err := r.Client.ServiceGroup(ctx, &configMap); err != nil {
			return nil, nil, errors.Wrapf(err, "unable to create kubernetes configMap for object %s", obj.GetName())
		}

		// volume is an abstraction that is mountable by containers. Here we relate a volume with a configmap
		volumes := []corev1.Volume{
			{
				Name: fmt.Sprintf("%s-volume", configMap.GetName()),
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: configMap.GetName()},
					},
				},
			},
		}

		// mounts are relations between the content of a volume and a directory on the container
		mounts := make([]corev1.VolumeMount, 0, len(config))
		for _, conf := range config {
			mounts = append(mounts, corev1.VolumeMount{
				Name:      volumes[0].Name,
				ReadOnly:  true,
				MountPath: conf.DstDir,
				SubPath:   conf.File,
			})
		}

		return volumes, mounts, nil

	*/

	return nil, nil, nil
}
