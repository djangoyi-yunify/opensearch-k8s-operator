package logstash

import (
	"context"
	"errors"

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/banzaicloud/operator-tools/pkg/reconciler"
	"github.com/go-logr/logr"
	opsterv1 "opensearch.opster.io/api/v1"
	"opensearch.opster.io/pkg/logstash/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ServiceReconciler struct {
	client.Client
	reconciler.ResourceReconciler
	ctx      context.Context
	instance *opsterv1.Logstash
	logger   logr.Logger
}

func NewServiceReconciler(
	client client.Client,
	ctx context.Context,
	instance *opsterv1.Logstash,
	opts ...reconciler.ResourceReconcilerOption,
) *ServiceReconciler {
	return &ServiceReconciler{
		Client: client,
		ResourceReconciler: reconciler.NewReconcilerWith(client,
			append(
				opts,
				reconciler.WithPatchCalculateOptions(patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(), patch.IgnoreStatusFields()),
				reconciler.WithLog(log.FromContext(ctx).WithValues("logstash subcontroller", "service")),
			)...,
		),
		ctx:      ctx,
		instance: instance,
		logger:   log.FromContext(ctx),
	}
}

func (r *ServiceReconciler) Reconcile() (ctrl.Result, error) {
	r.logger.Info("Reconciling service")

	if len(r.instance.Spec.Config.Ports) == 0 {
		err := errors.New("ports are needed")
		return ctrl.Result{}, err
	}

	result := reconciler.CombinedResult{}
	lstservice := utils.BuildService(r.instance)
	result.CombineErr(ctrl.SetControllerReference(r.instance, lstservice, r.Scheme()))
	result.Combine(r.ReconcileResource(lstservice, reconciler.StatePresent))
	return result.Result, result.Err
}
