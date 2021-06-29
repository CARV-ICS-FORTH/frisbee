package router

import (
	"fmt"

	"github.com/fnikolai/frisbee/pkg/router/endpoint"
	log "github.com/sirupsen/logrus"
)

type InnerObject interface{}

type routeEntry struct {
	// Name is the name of the handler
	Name string

	// Object is the type of objects the handler can handle (e.g, ServiceGroup, Stop, ...)
	Object InnerObject

	// Endpoint points to the actual implementation of the handler
	Endpoint endpoint.Handler
}

var routeTable map[string]*routeEntry

// Register registers an endpoint.
func Register(name string, obj InnerObject, newEndpoint endpoint.Handler) {
	_, ok := routeTable[name]
	if ok {
		panic(fmt.Sprintf("duplicate register for endpoint %s", name))
	}

	routeTable[name] = &routeEntry{
		Name:     name,
		Object:   obj,
		Endpoint: newEndpoint,
	}

	log.Debugf("Register %s endpoint", name)
}

func Get(kind string) endpoint.Handler {
	handler, ok := routeTable[kind]

	if !ok {
		fmt.Print("Available handlers: ", routeTable)
		panic(fmt.Sprintf("no handler is registered for kind %s", kind))
	}

	return handler.Endpoint
}

func init() {
	routeTable = make(map[string]*routeEntry)

	log.Print("Start routing table")
}

/*
func FetchChaosByTemplateType(templateType TemplateType) (runtime.Object, error) {
	if kind, ok := all.kinds[string(templateType)]; ok {
		return kind.Chaos.DeepCopyObject(), nil
	}
	return nil, fmt.Errorf("no such kind refers to template type %s", templateType)
}

*/
