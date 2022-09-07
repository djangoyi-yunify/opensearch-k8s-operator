package logstash

import (
	"context"

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/banzaicloud/operator-tools/pkg/reconciler"
	"github.com/go-logr/logr"
	opsterv1 "opensearch.opster.io/api/v1"
	"opensearch.opster.io/pkg/logstash/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ConfigMapReconciler struct {
	client.Client
	reconciler.ResourceReconciler
	ctx      context.Context
	instance *opsterv1.Logstash
	logger   logr.Logger
}

func NewConfigMapReconciler(
	client client.Client,
	ctx context.Context,
	instance *opsterv1.Logstash,
	opts ...reconciler.ResourceReconcilerOption,
) *ConfigMapReconciler {
	return &ConfigMapReconciler{
		Client: client,
		ResourceReconciler: reconciler.NewReconcilerWith(client,
			append(
				opts,
				reconciler.WithPatchCalculateOptions(patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(), patch.IgnoreStatusFields()),
				reconciler.WithLog(log.FromContext(ctx).WithValues("logstash subcontroller", "configmap")),
			)...,
		),
		ctx:      ctx,
		instance: instance,
		logger:   log.FromContext(ctx),
	}
}

func (r *ConfigMapReconciler) Reconcile() (ctrl.Result, error) {
	r.logger.Info("Reconciling configmap")

	result := reconciler.CombinedResult{}
	lstconfigmap := utils.BuildConfigMap(r.instance)
	result.CombineErr(ctrl.SetControllerReference(r.instance, lstconfigmap, r.Scheme()))
	result.Combine(r.ReconcileResource(lstconfigmap, reconciler.StatePresent))
	return result.Result, result.Err
}
