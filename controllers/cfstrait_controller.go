/*


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
	"sync"

	"github.com/ghostbaby/cfs-trait/controllers/models"

	"sigs.k8s.io/controller-runtime/pkg/source"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/client-go/tools/record"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	traitv1 "github.com/ghostbaby/cfs-trait/api/v1"
)

// CfsTraitReconciler reconciles a CfsTrait object
type CfsTraitReconciler struct {
	client.Client
	CTX        context.Context
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	KnownAlert *sync.Map
}

// +kubebuilder:rbac:groups=trait.ghostbaby.com,resources=cfstraits,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=trait.ghostbaby.com,resources=cfstraits/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods;configmaps;services;events;secret,verbs=get;list;watch;create;update;patch;delete

func (r *CfsTraitReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("cfstrait", req.NamespacedName)
	r.CTX = ctx

	log.Info("start to reconcile.")

	var cfs traitv1.CfsTrait
	if err := r.Get(ctx, req.NamespacedName, &cfs); err != nil {
		log.Error(err, "unable to fetch CfsTrait")
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.CallBrokerToExec(&cfs, req); err != nil {
		return reconcile.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *CfsTraitReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &corev1.Pod{}, "spec.nodeName",
		func(rawObj runtime.Object) []string {
			pod := rawObj.(*corev1.Pod)
			return []string{pod.Spec.NodeName}
		}); err != nil {
		return err
	}

	mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {

			pod, ok := a.Object.(*corev1.Pod)
			if !ok {
				return nil

			}

			if !IsPodReady(pod) {
				return nil
			}

			labels := a.Meta.GetLabels()
			appName, isSet := labels[models.DefaultAppLabelKey]
			if !isSet {
				return nil
			}

			isExist, namespacedName, err := r.IsPodCfsPolicyExist(appName, a.Meta.GetNamespace())
			if err != nil {
				return nil
			}

			if !isExist {
				return nil
			}

			return []reconcile.Request{
				{NamespacedName: namespacedName},
			}
		})

	return ctrl.NewControllerManagedBy(mgr).
		For(&traitv1.CfsTrait{}).
		Owns(&corev1.Pod{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}).
		Complete(r)
}

func updateObservedGeneration(w *traitv1.CfsTrait) {
	if w.Status.ObservedGeneration != w.Generation {
		w.Status.ObservedGeneration = w.Generation
	}
}

func isNeedUpdateStatus(w *traitv1.CfsTrait) bool {
	return w.Status.ObservedGeneration != w.Generation
}
