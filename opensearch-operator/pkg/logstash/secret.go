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

type SecretReconciler struct {
	client.Client
	reconciler.ResourceReconciler
	ctx      context.Context
	instance *opsterv1.Logstash
	logger   logr.Logger
}

func NewSecretReconciler(
	client client.Client,
	ctx context.Context,
	instance *opsterv1.Logstash,
	opts ...reconciler.ResourceReconcilerOption,
) *SecretReconciler {
	return &SecretReconciler{
		Client: client,
		ResourceReconciler: reconciler.NewReconcilerWith(client,
			append(
				opts,
				reconciler.WithPatchCalculateOptions(patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(), patch.IgnoreStatusFields()),
				reconciler.WithLog(log.FromContext(ctx).WithValues("logstash subcontroller", "secret")),
			)...,
		),
		ctx:      ctx,
		instance: instance,
		logger:   log.FromContext(ctx),
	}
}

func (r *SecretReconciler) Reconcile() (ctrl.Result, error) {
	r.logger.Info("Reconciling secret")

	if r.instance.Spec.Config.OpenSearchClusterRef == nil {
		r.logger.Info("Not define OpenSearchClusterName, not create secret")
		return ctrl.Result{}, nil
	}

	result := reconciler.CombinedResult{}
	lstsecret := utils.BuildSecret(r.instance)
	result.CombineErr(ctrl.SetControllerReference(r.instance, lstsecret, r.Scheme()))
	result.Combine(r.ReconcileResource(lstsecret, reconciler.StatePresent))
	return result.Result, result.Err
}
