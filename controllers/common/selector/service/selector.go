package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var Client client.Client

const (
	idListSeparator = " "
)

type ServiceList []v1alpha1.Service

func (list ServiceList) String() string {
	names := make([]string, len(list))

	for i, service := range list {
		names[i] = service.GetName()
	}

	return strings.Join(names, idListSeparator)
}

func IsMacro(macro string) bool {
	return strings.HasPrefix(macro, ".")
}

// ParseMacro translates a given macro to the appropriate selector and executes it.
// If the input is not a macro, it will be returned immediately as the first element of a string slice.
func ParseMacro(macro string) *v1alpha1.ServiceSelector {
	var criteria v1alpha1.ServiceSelector

	fields := strings.Split(macro, ".")

	if len(fields) != 4 {
		panic(errors.Errorf("%s is not a valid macro", macro))
	}

	kind := fields[1]
	object := fields[2]
	filter := fields[3]

	switch kind {
	case "servicegroup":
		criteria.Selector.ServiceGroup = object
		criteria.Mode = v1alpha1.Mode(filter)

		return &criteria

	default:
		panic(errors.Errorf("%s is not a valid macro", macro))
	}
}

func Select(ctx context.Context, ss *v1alpha1.ServiceSelector) ServiceList {
	if ss == nil {
		logrus.Warn("empty service selector")
		return []v1alpha1.Service{}
	}

	// get all available services that match the criteria
	services, err := selectServices(ctx, &ss.Selector)
	if err != nil {
		logrus.Warn(err)

		return []v1alpha1.Service{}
	}

	if len(services) == 0 {
		return []v1alpha1.Service{}
	}

	// filter services based on the pods
	filteredServices, err := filterServicesByMode(services, ss.Mode, ss.Value)
	if err != nil {
		logrus.Warn(err)

		return []v1alpha1.Service{}
	}

	return filteredServices
}

func selectServices(ctx context.Context, ss *v1alpha1.ServiceSelectorSpec) ([]v1alpha1.Service, error) {
	var services []v1alpha1.Service

	appendService := func(ns, name string) error {
		var service v1alpha1.Service

		key := client.ObjectKey{
			Namespace: ns,
			Name:      name,
		}

		if err := Client.Get(ctx, key, &service); err != nil {
			return errors.Wrapf(err, "unable to find %s", key)
		}

		services = append(services, service)

		return nil
	}

	// case 1. services are specifically specified
	if len(ss.Services) > 0 {
		for ns, names := range ss.Services {
			for _, name := range names {
				if err := appendService(ns, name); err != nil {
					return nil, err
				}
			}
		}
	}

	// case 2. servicegroups are specifically specified
	listOptions := client.ListOptions{}

	if len(ss.ServiceGroup) > 0 || len(ss.Labels) > 0 {

		if len(ss.ServiceGroup) > 0 {
			ss.Labels = labels.Merge(ss.Labels, map[string]string{
				"owner": ss.ServiceGroup,
			})
		}

		metav1Ls := &metav1.LabelSelector{
			MatchLabels: ss.Labels,
		}

		ls, err := metav1.LabelSelectorAsSelector(metav1Ls)
		if err != nil {
			return nil, err
		}

		listOptions.LabelSelector = ls
	}

	// select services For more options see
	// https://github.com/chaos-mesh/chaos-mesh/blob/31aef289b81a1d713b5a9976a257090da81ac29e/pkg/selector/pod/selector.go
	var serviceList v1alpha1.ServiceList

	if err := Client.List(ctx, &serviceList, &listOptions); err != nil {
		return nil, err
	}

	return serviceList.Items, nil
}

func filterServicesByMode(services []v1alpha1.Service, mode v1alpha1.Mode, value string) ([]v1alpha1.Service, error) {
	if len(services) == 0 {
		return nil, errors.New("cannot generate services from empty list")
	}

	switch mode {
	case v1alpha1.OneMode:
		index := getRandomNumber(len(services))
		service := services[index]

		return []v1alpha1.Service{service}, nil
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
			return nil, err
		}

		if percentage == 0 {
			return nil, errors.New("cannot select any pod as value below or equal 0")
		}

		if percentage < 0 || percentage > 100 {
			return nil, fmt.Errorf("fixed percentage value of %d is invalid, Must be (0,100]", percentage)
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
			return nil, fmt.Errorf("fixed percentage value of %d is invalid, Must be [0-100]", maxPercentage)
		}

		percentage := getRandomNumber(maxPercentage + 1) // + 1 because Intn works with half open interval [0,n) and we want [0,n]
		num := int(math.Floor(float64(len(services)) * float64(percentage) / 100))

		return getFixedSubListFromServiceList(services, num), nil
	default:
		return nil, fmt.Errorf("mode %s not supported", mode)
	}
}

func getRandomNumber(max int) uint64 {
	num, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return num.Uint64()
}

func getFixedSubListFromServiceList(services []v1alpha1.Service, num int) []v1alpha1.Service {
	indexes := RandomFixedIndexes(0, uint(len(services)), uint(num))

	var filteredServices []v1alpha1.Service

	for _, index := range indexes {
		index := index
		filteredServices = append(filteredServices, services[index])
	}

	return filteredServices
}

// RandomFixedIndexes returns the `count` random indexes between `start` and `end`.
// [start, end)
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
