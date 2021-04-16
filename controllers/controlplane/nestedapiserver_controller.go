/*
Copyright 2021 The Kubernetes Authors.

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

package controlplane

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcli "sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1 "sigs.k8s.io/cluster-api-provider-nested/apis/controlplane/v1alpha4"
)

// NestedAPIServerReconciler reconciles a NestedAPIServer object
type NestedAPIServerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=nestedapiservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=nestedapiservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=nestedapiservers/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulset,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulset/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=,resources=service,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=,resources=service/status,verbs=get;update;patch

func (r *NestedAPIServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("nestedapiserver", req.NamespacedName)
	log.Info("Reconciling NestedAPIServer...")
	var nkas clusterv1.NestedAPIServer
	if err := r.Get(ctx, req.NamespacedName, &nkas); err != nil {
		return ctrl.Result{}, ctrlcli.IgnoreNotFound(err)
	}
	log.Info("creating NestedAPIServer",
		"namespace", nkas.GetNamespace(),
		"name", nkas.GetName())

	// 1. check if the ownerreference has been set by the
	// NestedControlPlane controller.
	owner := getOwner(nkas.ObjectMeta)
	if owner == (metav1.OwnerReference{}) {
		// requeue the request if the owner NestedControlPlane has
		// not been set yet.
		log.Info("the owner has not been set yet, will retry later",
			"namespace", nkas.GetNamespace(),
			"name", nkas.GetName())
		return ctrl.Result{Requeue: true}, nil
	}

	// 2. create the NestedAPIServer StatefulSet if not found
	var nkasSts appsv1.StatefulSet
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: nkas.GetNamespace(),
		Name:      nkas.GetName(),
	}, &nkasSts); err != nil {
		if apierrors.IsNotFound(err) {
			// as the statefulset is not found, mark the NestedAPIServer as unready
			if IsComponentReady(nkas.Status.CommonStatus) {
				nkas.Status.Phase =
					string(clusterv1.Unready)
				log.V(5).Info("The corresponding statefulset is not found, " +
					"will mark the NestedAPIServer as unready")
				if err := r.Status().Update(ctx, &nkas); err != nil {
					log.Error(err, "fail to update the status of the NestedAPIServer Object")
					return ctrl.Result{}, err
				}
			}
			// the statefulset is not found, create one
			if err := createNestedComponentSts(ctx,
				r.Client, nkas.ObjectMeta, nkas.Spec.NestedComponentSpec,
				clusterv1.APIServer, owner.Name, log); err != nil {
				log.Error(err, "fail to create NestedAPIServer StatefulSet")
				return ctrl.Result{}, err
			}
			log.Info("successfully create the NestedAPIServer StatefulSet")
			return ctrl.Result{}, nil
		}
		log.Error(err, "fail to get NestedAPIServer StatefulSet")
		return ctrl.Result{}, err
	}

	// 3. reconcile the NestedAPIServer based on the status of the StatefulSet.
	// Mark the NestedAPIServer as Ready if the StatefulSet is ready
	if nkasSts.Status.ReadyReplicas == nkasSts.Status.Replicas {
		log.Info("The NestedAPIServer StatefulSet is ready")
		if IsComponentReady(nkas.Status.CommonStatus) {
			// As the NestedAPIServer StatefulSet is ready, update
			// NestedAPIServer status
			nkas.Status.Phase = string(clusterv1.Ready)
			objRef, err := genAPIServerSvcRef(r.Client, nkas)
			if err != nil {
				log.Error(err, "fail to generate NestedAPIServer Service Reference")
				return ctrl.Result{}, err
			}
			nkas.Status.APIServerService = &objRef

			log.V(5).Info("The corresponding statefulset is ready, " +
				"will mark the NestedAPIServer as ready")
			if err := r.Status().Update(ctx, &nkas); err != nil {
				log.Error(err, "fail to update NestedAPIServer Object")
				return ctrl.Result{}, err
			}
			log.Info("Successfully set the NestedAPIServer object to ready")
		}
		return ctrl.Result{}, nil
	}

	// mark the NestedAPIServer as unready, if the NestedAPIServer
	// StatefulSet is unready,
	if IsComponentReady(nkas.Status.CommonStatus) {
		nkas.Status.Phase = string(clusterv1.Unready)
		if err := r.Status().Update(ctx, &nkas); err != nil {
			log.Error(err, "fail to update NestedAPIServer Object")
			return ctrl.Result{}, err
		}
		log.Info("Successfully set the NestedAPIServer object to unready")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NestedAPIServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(),
		&appsv1.StatefulSet{},
		statefulsetOwnerKeyNKas,
		func(rawObj ctrlcli.Object) []string {
			// grab the statefulset object, extract the owner
			sts := rawObj.(*appsv1.StatefulSet)
			owner := metav1.GetControllerOf(sts)
			if owner == nil {
				return nil
			}
			// make sure it's a NestedAPIServer
			if owner.APIVersion != clusterv1.GroupVersion.String() ||
				owner.Kind != string(clusterv1.APIServer) {
				return nil
			}

			// and if so, return it
			return []string{owner.Name}
		}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.NestedAPIServer{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
