package logstash

import (
	"context"

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/banzaicloud/operator-tools/pkg/reconciler"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	opsterv1 "opensearch.opster.io/api/v1"
	"opensearch.opster.io/pkg/logstash/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type OsInfoReconciler struct {
	client.Client
	reconciler.ResourceReconciler
	ctx      context.Context
	instance *opsterv1.Logstash
	logger   logr.Logger
}

func NewOsInfoReconciler(
	client client.Client,
	ctx context.Context,
	instance *opsterv1.Logstash,
	opts ...reconciler.ResourceReconcilerOption,
) *OsInfoReconciler {
	return &OsInfoReconciler{
		Client: client,
		ResourceReconciler: reconciler.NewReconcilerWith(client,
			append(
				opts,
				reconciler.WithPatchCalculateOptions(patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(), patch.IgnoreStatusFields()),
				reconciler.WithLog(log.FromContext(ctx).WithValues("logstash subcontroller", "osinfo")),
			)...,
		),
		ctx:      ctx,
		instance: instance,
		logger:   log.FromContext(ctx),
	}
}

func (r *OsInfoReconciler) Reconcile() (ctrl.Result, error) {
	r.logger.Info("Reconciling osinfo")

	if r.instance.Spec.Config.OpenSearchClusterRef == nil {
		r.logger.Info("Not define OpenSearchClusterName")
		return ctrl.Result{}, nil
	}

	if len(r.instance.Spec.Config.OpenSearchClusterRef.Name) == 0 || len(r.instance.Spec.Config.OpenSearchClusterRef.Service) == 0 {
		r.logger.Info("crd didn't provide full openSearchClusterRef info, can not calculate its service info")
		return ctrl.Result{}, nil
	}

	var ns string
	if len(r.instance.Spec.Config.OpenSearchClusterRef.Namespace) == 0 {
		ns = r.instance.Namespace
	} else {
		ns = r.instance.Spec.Config.OpenSearchClusterRef.Namespace
	}

	utils.BuildExtOpenSearchUrl(r.instance.Spec.Config.OpenSearchClusterRef.Service, ns)

	if len(r.instance.Spec.Config.OpenSearchClusterRef.Secret) == 0 {
		r.logger.Info("crd didn't provide opensearch secret name, can not fetch its secret info")
		return ctrl.Result{}, nil
	}

	tmpsec := &corev1.Secret{}
	if err := r.Get(r.ctx, types.NamespacedName{Namespace: ns, Name: r.instance.Spec.Config.OpenSearchClusterRef.Secret}, tmpsec); err != nil {
		r.logger.Info("can not fetch secret info")
		return ctrl.Result{}, err
	}
	utils.ExtOpenSearchLogstashUserSecret = tmpsec

	return ctrl.Result{}, nil
}
