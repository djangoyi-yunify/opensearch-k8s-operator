package logstash

import (
	"context"

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/banzaicloud/operator-tools/pkg/reconciler"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	opsterv1 "opensearch.opster.io/api/v1"
	"opensearch.opster.io/pkg/logstash/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type StatusReconciler struct {
	client.Client
	reconciler.ResourceReconciler
	ctx      context.Context
	instance *opsterv1.Logstash
	logger   logr.Logger
}

func NewStatusReconciler(
	client client.Client,
	ctx context.Context,
	instance *opsterv1.Logstash,
	opts ...reconciler.ResourceReconcilerOption,
) *StatusReconciler {
	return &StatusReconciler{
		Client: client,
		ResourceReconciler: reconciler.NewReconcilerWith(client,
			append(
				opts,
				reconciler.WithPatchCalculateOptions(patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(), patch.IgnoreStatusFields()),
				reconciler.WithLog(log.FromContext(ctx).WithValues("logstash subcontroller", "status")),
			)...,
		),
		ctx:      ctx,
		instance: instance,
		logger:   log.FromContext(ctx),
	}
}

func (r *StatusReconciler) Reconcile() (ctrl.Result, error) {
	r.logger.Info("Reconciling status")

	dl := &appsv1.Deployment{}
	if err := r.Get(r.ctx, types.NamespacedName{Namespace: r.instance.Namespace, Name: utils.GetDeploymentName(r.instance.Name)}, dl); err != nil {
		return ctrl.Result{}, err
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if dl.Status.ReadyReplicas == dl.Status.Replicas {
			r.instance.Status.Phase = opsterv1.LogstashPhaseRunning
		} else {
			r.instance.Status.Phase = opsterv1.LogstashPhasePending
		}
		return r.Status().Update(r.ctx, r.instance)
	})

	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
