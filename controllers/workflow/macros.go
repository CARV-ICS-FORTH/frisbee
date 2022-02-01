/*
Copyright 2021 ICS-FORTH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package workflow

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/carv-ics-forth/frisbee/api/v1alpha1"
	"github.com/carv-ics-forth/frisbee/controllers/utils"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	idListSeparator = " "
)

// SList is a service list
type SList []*v1alpha1.Service

func (in SList) ToString() string {
	if len(in) == 0 {
		return ""
	}

	return strings.Join(in.GetNames(), idListSeparator)
}

func (in SList) GetNames() []string {
	if len(in) == 0 {
		return nil
	}

	names := make([]string, len(in))

	for i, service := range in {
		names[i] = service.GetName()
	}

	return names
}

func isMacro(macro string) bool {
	return strings.HasPrefix(macro, ".")
}

func parseMacro(namespace string, ss *v1alpha1.ServiceSelector) error {
	fields := strings.Split(*ss.Macro, ".")

	if len(fields) != 4 {
		panic(errors.Errorf("%s is not a valid macro", *ss.Macro))
	}

	kind := fields[1]
	object := fields[2]
	filter := fields[3]

	switch kind {
	case "cluster":
		ss.Match.ByCluster = map[string]string{namespace: object}
		ss.Mode = v1alpha1.Convert(filter)

	case "service":
		ss.Match.ByName = map[string][]string{namespace: {object}}
		ss.Mode = v1alpha1.Convert(filter)

	default:
		return errors.Errorf("%v is not a valid macro", *ss.Macro)
	}

	return nil
}

func expandSliceInputs(ctx context.Context, s *Controller, nm string, inputs *[]string) error {
	if inputs == nil || *inputs == nil {
		return nil
	}

	// cache the results of macro as to avoid asking the Kubernetes API. This, however, is only applicable
	// to the level of a caller, because different groups may be created in different moments
	// throughout the experiment, thus yielding different results.
	cache := make(map[string]SList)

	// extend macros
	for i, value := range *inputs {
		if isMacro(value) {

			val := value
			ss := &v1alpha1.ServiceSelector{Macro: &val}

			if err := parseMacro(nm, ss); err != nil {
				return errors.Wrapf(err, "input [%d]", i)
			}

			services, exists := cache[value]
			if !exists {
				runningServices, err := selectServices(ctx, s, &ss.Match)
				if err != nil {
					return errors.Wrapf(err, "service selection error")
				}

				if len(runningServices) == 0 {
					// it is possible that some services exist, but they are not in the Running phase.
					// In this case, we should retry getting the services.
					return errors.Errorf("macro %s yields no services", value)
				}

				services = runningServices
				cache[value] = runningServices
			}

			// filter services based on the pods
			filteredServices, err := filterByMode(services, ss.Mode, ss.Value)
			if err != nil {
				return errors.Wrapf(err, "filter by mode")
			}

			(*inputs)[i] = filteredServices.ToString()
		}
	}

	return nil
}

func expandMapInputs(ctx context.Context, s *Controller, nm string, inputs *[]map[string]string) error {
	if inputs == nil || *inputs == nil {
		return nil
	}

	// cache the results of macro as to avoid asking the Kubernetes API. This, however, is only applicable
	// to the level of a caller, because different groups may be created in different moments
	// throughout the experiment, thus yielding different results.
	cache := make(map[string]SList)

	// extend macros
	for i, input := range *inputs {
		for key, value := range input {
			if isMacro(value) {

				val := value
				ss := &v1alpha1.ServiceSelector{Macro: &val}

				if err := parseMacro(nm, ss); err != nil {
					return errors.Wrapf(err, "input [%d]", i)
				}

				services, exists := cache[value]
				if !exists {
					runningServices, err := selectServices(ctx, s, &ss.Match)
					if err != nil {
						return errors.Wrapf(err, "service selection error")
					}

					if len(runningServices) == 0 {
						// it is possible that some services exist, but they are not in the Running phase.
						// In this case, we should retry getting the services.
						return errors.Errorf("macro %s yields no services", value)
					}

					services = runningServices
					cache[value] = runningServices
				}

				// filter services based on the pods
				filteredServices, err := filterByMode(services, ss.Mode, ss.Value)
				if err != nil {
					return errors.Wrapf(err, "filter by mode")
				}

				(*inputs)[i][key] = filteredServices.ToString()
			}
		}
	}

	return nil
}

func selectServices(ctx context.Context, r utils.Reconciler, ss *v1alpha1.MatchBy) (SList, error) {
	if ss == nil {
		return nil, nil
	}

	var serviceList SList

	// case 1. select services by namespace and by name.
	if len(ss.ByName) > 0 {
		for ns, names := range ss.ByName {
			for _, name := range names {
				var service v1alpha1.Service

				key := client.ObjectKey{
					Namespace: ns,
					Name:      name,
				}

				if err := r.GetClient().Get(ctx, key, &service); err != nil {
					return nil, errors.Wrapf(err, "unable to find service %s", key)
				}

				// use only running services
				if service.Status.Lifecycle.Phase == v1alpha1.PhaseRunning {
					serviceList = append(serviceList, &service)
				}
			}
		}
	}

	// case 2. select services by they clusterName they belong to.
	for nm, clusterName := range ss.ByCluster {
		key := client.ObjectKey{ // search all
			Namespace: nm,
			Name:      clusterName,
		}

		{
			var cluster v1alpha1.Cluster

			if err := r.GetClient().Get(ctx, key, &cluster); err != nil {
				return nil, errors.Wrapf(err, "cannot find cluster %s", key)
			}

			var slist v1alpha1.ServiceList

			err := r.GetClient().List(ctx, &slist, client.MatchingLabels{v1alpha1.LabelManagedBy: cluster.GetName()})
			if err != nil {
				return nil, errors.Wrapf(err, "cannot get services")
			}

			// use only the running services
			for i, service := range slist.Items {
				if service.Status.Lifecycle.Phase == v1alpha1.PhaseRunning {
					serviceList = append(serviceList, &slist.Items[i])
				}
			}
		}
	}

	/*
		// case 3. labels
		var listOptions client.ListOptions

		if len(ss.Labels) > 0 {
			ls, err := metav1.LabelSelectorAsSelector(metav1.SetAsLabelSelector(ss.Labels))
			if err != nil {
				return nil, err
			}

			listOptions = client.ListOptions{LabelSelector: ls}
		}

		var podList corev1.PodList

			// case 4. ByNamespace
			if len(ss.Namespaces) > 0 { // search specified namespaces
				for _, namespace := range ss.Namespaces {
					listOptions.Namespace = namespace

					if err := common.Globals.Client.List(ctx, &serviceList, &listOptions); err != nil {
						return nil, err
					}

					services = append(services, serviceList.Items...)
				}
			} else { // search all namespaces
				if err := common.Globals.Client.List(ctx, &serviceList, &listOptions); err != nil {
					return nil, errors.Wrapf(err, "namespace error")
				}

				services = append(services, serviceList.Items...)
			}

	*/

	// select services For more options see
	// https://github.com/chaos-mesh/chaos-mesh/blob/31aef289b81a1d713b5a9976a257090da81ac29e/pkg/selector/pod/selector.go

	return serviceList, nil
}

func filterByMode(services SList, mode v1alpha1.Mode, value string) (SList, error) {
	if len(services) == 0 {
		return nil, errors.New("cannot generate services from empty list")
	}

	switch mode {
	case v1alpha1.OneMode:
		index := getRandomNumber(len(services))
		service := services[index]

		return SList{service}, nil
	case v1alpha1.AllMode:
		return services, nil

	case v1alpha1.FixedMode:
		num, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}

		if len(services) < num {
			num = len(services)
		}

		if num <= 0 {
			return nil, errors.New("cannot select any service as value below or equal 0")
		}

		return getFixedSubListFromServiceList(services, num), nil
	case v1alpha1.FixedPercentMode:
		percentage, err := strconv.Atoi(value)
		if err != nil {
			return nil, errors.Wrapf(err, "conversion error")
		}

		if percentage == 0 {
			return nil, errors.New("cannot select any pod as value below or equal 0")
		}

		if percentage < 0 || percentage > 100 {
			return nil, errors.Errorf("fixed percentage value of %d is invalid, Must be (0,100]", percentage)
		}

		num := int(math.Floor(float64(len(services)) * float64(percentage) / 100))

		return getFixedSubListFromServiceList(services, num), nil
	case v1alpha1.RandomMaxPercentMode:
		maxPercentage, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}

		if maxPercentage == 0 {
			return nil, errors.New("cannot select any pod as value below or equal 0")
		}

		if maxPercentage < 0 || maxPercentage > 100 {
			return nil, errors.Errorf("fixed percentage value of %d is invalid, Must be [0-100]", maxPercentage)
		}

		// + 1 because Intn works with half open interval [0,n) and we want [0,n]
		percentage := getRandomNumber(maxPercentage + 1)
		num := int(math.Floor(float64(len(services)) * float64(percentage) / 100))

		return getFixedSubListFromServiceList(services, num), nil
	default:
		return nil, errors.Errorf("mode %s not supported", mode)
	}
}

func getRandomNumber(max int) uint64 {
	num, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))

	return num.Uint64()
}

func getFixedSubListFromServiceList(services SList, num int) SList {
	indexes := RandomFixedIndexes(0, uint(len(services)), uint(num))

	var filteredServices SList

	for _, index := range indexes {
		filteredServices = append(filteredServices, services[index])
	}

	return filteredServices
}

// RandomFixedIndexes returns the `count` random indexes between `start` and `end`.
// [start, end).
func RandomFixedIndexes(start, end, count uint) []uint {
	var indexes []uint

	m := make(map[uint]uint, count)

	if end < start {
		return indexes
	}

	if count > end-start {
		for i := start; i < end; i++ {
			indexes = append(indexes, i)
		}

		return indexes
	}

	for i := 0; i < int(count); {
		index := uint(getRandomNumber(int(end-start))) + start

		_, exist := m[index]
		if exist {
			continue
		}

		m[index] = index
		indexes = append(indexes, index)
		i++
	}

	return indexes
}
