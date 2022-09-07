/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	opsterv1 "opensearch.opster.io/api/v1"
	"opensearch.opster.io/pkg/helpers"
	lst "opensearch.opster.io/pkg/logstash"
)

// LogstashReconciler reconciles a Logstash object
type LogstashReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Instance *opsterv1.Logstash
	logr.Logger
}

//+kubebuilder:rbac:groups=opster.opensearch.opster.io,resources=logstashes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=opster.opensearch.opster.io,resources=logstashes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=opster.opensearch.opster.io,resources=logstashes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Logstash object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *LogstashReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Logger = log.FromContext(ctx).WithValues("logstash", req.NamespacedName)
	r.Logger.Info("Reconciling Logstash")

	myFinalizerName := "Opster"

	r.Instance = &opsterv1.Logstash{}
	err := r.Get(ctx, req.NamespacedName, r.Instance)
	if err != nil {
		r.Logger.Error(err, "unable to fetch Logsash")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if r.Instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !helpers.ContainsString(r.Instance.GetFinalizers(), myFinalizerName) {
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := r.Get(ctx, req.NamespacedName, r.Instance); err != nil {
					return err
				}
				controllerutil.AddFinalizer(r.Instance, myFinalizerName)
				return r.Update(ctx, r.Instance)
			})
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if helpers.ContainsString(r.Instance.GetFinalizers(), myFinalizerName) {
			if result, err := r.deleteExternalResources(ctx, req); err != nil {
				return result, err
			}
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := r.Get(ctx, req.NamespacedName, r.Instance); err != nil {
					return err
				}
				controllerutil.RemoveFinalizer(r.Instance, myFinalizerName)
				return r.Update(ctx, r.Instance)
			})
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if len(r.Instance.Status.Phase) == 0 {
		r.Instance.Status.Phase = opsterv1.LogstashPhasePending
		if err := r.Update(ctx, r.Instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	result, err := r.internalReconcile(ctx, req)
	if err != nil {
		return result, err
	}

	result, err = r.updateStatus(ctx, req)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *LogstashReconciler) deleteExternalResources(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *LogstashReconciler) internalReconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	sec := lst.NewSecretReconciler(r.Client, ctx, r.Instance)
	res, err := sec.Reconcile()
	return res, err
}

func (r *LogstashReconciler) updateStatus(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LogstashReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opsterv1.Logstash{}).
		Complete(r)
}
