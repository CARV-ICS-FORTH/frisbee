package common

import (
	"github.com/fnikolai/frisbee/controllers/common/selector/service"
	"github.com/fnikolai/frisbee/controllers/common/selector/template"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var common struct {
	cache  cache.Cache
	client client.Client
	logger logr.Logger
}

func InitCommon(mgr ctrl.Manager, logger logr.Logger) {
	common.cache = mgr.GetCache()
	common.client = mgr.GetClient()
	common.logger = logger.WithName("common")

	service.Client = common.client
	template.Client = common.client
}
