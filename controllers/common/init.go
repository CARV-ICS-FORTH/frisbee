package common

import (
	"reflect"

	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/fnikolai/frisbee/controllers/common/selector/template"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type watch struct {
	cache  cache.Cache
	client client.Client
	logger logr.Logger

	selectorMap map[reflect.Type]interface{}
}

var common watch

func InitCommon(mgr ctrl.Manager, logger logr.Logger) {
	common.cache = mgr.GetCache()
	common.client = mgr.GetClient()
	common.logger = mgr.GetLogger()

	service.Client = common.client
	template.Client = common.client
}
