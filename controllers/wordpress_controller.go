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
	"github.com/James-Moore/wordpress-operator/controllers/common"
	"github.com/James-Moore/wordpress-operator/controllers/deployment"
	"github.com/James-Moore/wordpress-operator/controllers/pod"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	//"reflect"
	wordpressfullstackv1 "github.com/James-Moore/wordpress-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *WordpressReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	//r.Common.Log = r.Common.Log.WithValues("wordpress", req.NamespacedName)

	r.Common.Log.Info("CHECKPOINT 1")
	r.Common.Ctx = context.Background()
	r.Common.Wordpress = &wordpressfullstackv1.Wordpress{}

	breakControl, err := r.Common.Init(req.NamespacedName)
	if breakControl {
		return ctrl.Result{}, err
	}

	r.Common.Log.Info("CHECKPOINT 2")
	deploymentManager := &deployment.DeploymentManager{C: r.Common}
	breakControl, err = deploymentManager.Reconcile()
	if breakControl {
		return ctrl.Result{}, err
	}

	r.Common.Log.Info("CHECKPOINT 3")
	podManager := &pod.PodManager{Common: r.Common}
	breakControl, err = podManager.Reconcile()
	if breakControl {
		return ctrl.Result{}, err
	}

	// Update status.Nodes if needed
	//if !reflect.DeepEqual(podNames, wordpress.Status.Nodes) {
	//	wordpress.Status.Nodes = podNames
	//	err := r.Status().Update(ctx, wordpress)
	//	if err != nil {
	//		log.Error(err, "Failed to update Wordpress status")
	//		return ctrl.Result{}, err
	//	}
	//}

	r.Common.Log.Info("CHECKPOINT 4")
	return ctrl.Result{}, nil
}

// deploymentForWordpress returns a wordpress Deployment object
func updateDeploymentDefinition(deployment *appsv1.Deployment, container corev1.Container) {
	containers := deployment.Spec.Template.Spec.Containers
	deployment.Spec.Template.Spec.Containers = append(containers, container)
}

func (r *WordpressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//builder := ctrl.NewControllerManagedBy(mgr)
	//builder = builder.For(&wordpressfullstackv1.Wordpress{})
	//builder = builder.Owns(&appsv1.Deployment{})
	//return builder.Complete(r)

	return ctrl.NewControllerManagedBy(mgr).
		For(&wordpressfullstackv1.Wordpress{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)

}

// +kubebuilder:rbac:groups=wordpress-fullstack.jamesmoore.in,resources=wordpresss,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wordpress-fullstack.jamesmoore.in,resources=wordpresss/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

// WordpressReconciler reconciles a Wordpress object
type WordpressReconciler struct {
	Common *common.Common
}
