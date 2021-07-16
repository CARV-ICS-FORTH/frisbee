package dataport

import (
	"context"

	"github.com/fnikolai/frisbee/api/v1alpha1"
	"github.com/pkg/errors"
)

type kafka struct {
	r *Reconciler
}

func (p *kafka) Create(ctx context.Context, obj *v1alpha1.DataPort) error {
	return errors.New("not implemented")
}

func (p *kafka) Accept(ctx context.Context, obj *v1alpha1.DataPort) bool {
	p.r.Logger.Error(errors.New("not implemented"), "accept for kafka")

	return false
}
