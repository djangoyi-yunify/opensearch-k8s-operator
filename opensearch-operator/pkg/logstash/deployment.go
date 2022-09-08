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

type DeploymentReconciler struct {
	client.Client
	reconciler.ResourceReconciler
	ctx      context.Context
	instance *opsterv1.Logstash
	logger   logr.Logger
	hash     string
}

func NewDeploymentReconciler(
	client client.Client,
	ctx context.Context,
	instance *opsterv1.Logstash,
	hash string,
	opts ...reconciler.ResourceReconcilerOption,
) *DeploymentReconciler {
	return &DeploymentReconciler{
		Client: client,
		ResourceReconciler: reconciler.NewReconcilerWith(client,
			append(
				opts,
				reconciler.WithPatchCalculateOptions(patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(), patch.IgnoreStatusFields()),
				reconciler.WithLog(log.FromContext(ctx).WithValues("logstash subcontroller", "deployment")),
			)...,
		),
		ctx:      ctx,
		instance: instance,
		hash:     hash,
		logger:   log.FromContext(ctx),
	}
}

func (r *DeploymentReconciler) Reconcile() (ctrl.Result, error) {
	r.logger.Info("Reconciling deployment")

	result := reconciler.CombinedResult{}
	lstdeployment := utils.BuildDeployment(r.instance)
	result.CombineErr(ctrl.SetControllerReference(r.instance, lstdeployment, r.Scheme()))
	result.Combine(r.ReconcileResource(lstdeployment, reconciler.StatePresent))
	return result.Result, result.Err
}
