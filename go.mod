module github.com/fnikolai/frisbee

go 1.16

replace sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.9.0-beta.6

require (
	github.com/armon/go-radix v0.0.0-20180808171621-7fddfc383310
	github.com/davecgh/go-spew v1.1.1
	github.com/go-logr/logr v0.4.0
	github.com/go-playground/validator/v10 v10.6.1
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/pkg/errors v0.9.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.7.0
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	sigs.k8s.io/controller-runtime v0.0.0-00010101000000-000000000000
)
