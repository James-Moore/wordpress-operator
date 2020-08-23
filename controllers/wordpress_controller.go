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
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	//"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	wordpressfullstackv1 "github.com/James-Moore/wordpress-operator/api/v1"
)

func (r *WordpressReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("wordpress", req.NamespacedName)

	log.Info("CHECKPOINT 1")
	// Fetch the Wordpress instance
	wordpress := &wordpressfullstackv1.Wordpress{}
	err := r.Get(ctx, req.NamespacedName, wordpress)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Wordpress resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Wordpress")
		return ctrl.Result{}, err
	}

	log.Info("CHECKPOINT 2")
	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name, Namespace: wordpress.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		cntrs := r.createContainers(wordpress)
		// Define a new deployment
		dep := r.createDeployment(wordpress, cntrs)

		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	log.Info("CHECKPOINT 3")
	//JAMES TODO I BELIEVE THIS IS WHERE WE BUILD OUT THE DEPLOYMENT INCREMENTALLY.  STEP 1 CHECK FOR VOLUME IF NOT THERE DEPLOY.  STEP 2 CHECK FOR MYSQL IF NOT HERE CREATE.  STEP 3 SAME FOR WORDPRESS. STEP 4 SAME FOR SERVICE
	println(found.Spec.Template.Spec.Containers)
	// Ensure the deployment size is the same as the spec
	//size := wordpress.Spec.Size
	//if *found.Spec.Replicas != size {
	//	found.Spec.Replicas = &size
	//	//println(found.Spec.Template.Spec.Containers[0])
	//	err = r.Update(ctx, found)
	//	if err != nil {
	//		log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
	//		return ctrl.Result{}, err
	//	}
	//	// Spec updated - return and requeue
	//	return ctrl.Result{Requeue: true}, nil
	//}

	// Update the Wordpress status with the pod names
	log.Info("CHECKPOINT 4")

	// Define the values to match Pods against
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(wordpress.Namespace),
		client.MatchingLabels(labelsForWordpress(wordpress.Name)),
	}

	// Get the list of pods matching our search parameters: Namespace and Labels
	log.Info("CHECKPOINT 5")
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "Wordpress.Namespace", wordpress.Namespace, "Wordpress.Name", wordpress.Name)
		return ctrl.Result{}, err
	}

	log.Info("CHECKPOINT 6")
	// Get the names of the pods from our list
	podNames := getPodNames(podList.Items)

	for _, podName := range podNames {
		log.Info(podName)
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

	log.Info("CHECKPOINT 7")
	log.Info("CHECKPOINT 8")
	return ctrl.Result{}, nil
}

func (r *WordpressReconciler) createContainers(m *wordpressfullstackv1.Wordpress) []corev1.Container {
	mysqlContainer := r.createMySqlContainer(m)
	wordpressContainer := r.createWordpressContainer(m)
	return []corev1.Container{mysqlContainer, wordpressContainer}
}

func (r *WordpressReconciler) createWordpressContainer(m *wordpressfullstackv1.Wordpress) corev1.Container {
	wordpressContainer := corev1.Container{
		Name:  "wordpress",
		Image: "wordpress",
		Env: []corev1.EnvVar{
			{
				Name:  "WORDPRESS_DB_HOST",
				Value: "127.0.0.1:3306",
			},
			{
				Name:  "WORDPRESS_DB_USER",
				Value: "root",
			},
			{
				Name:  "WORDPRESS_DB_PASSWORD",
				Value: "Password1234",
			},
			{
				Name:  "WORDPRESS_DB_NAME",
				Value: "wordpress",
			},
			{
				Name:  "WORDPRESS_TABLE_PREFIX",
				Value: "wp_",
			},
		},
	}
	return wordpressContainer
}

func (r *WordpressReconciler) createMySqlContainer(m *wordpressfullstackv1.Wordpress) corev1.Container {
	mysqlContainer := corev1.Container{
		Name:  "mysql",
		Image: "mysql",
		Ports: []corev1.ContainerPort{{
			Name:          "mysql",
			ContainerPort: 3306,
			Protocol:      "TCP",
			HostIP:        "127.0.0.1",
			HostPort:      3306,
		}},
		Env: []corev1.EnvVar{{
			Name:  "MYSQL_ROOT_PASSWORD",
			Value: "Password1234",
		}},
	}

	return mysqlContainer
}

// deploymentForWordpress returns a wordpress Deployment object
func (r *WordpressReconciler) createDeployment(m *wordpressfullstackv1.Wordpress, containers []corev1.Container) *appsv1.Deployment {
	ls := labelsForDeployment(m.Name)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			//Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: containers,
				},
			},
		},
	}

	// Set Wordpress instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// deploymentForWordpress returns a wordpress Deployment object
func (r *WordpressReconciler) deploymentForWordpress(m *wordpressfullstackv1.Wordpress) *appsv1.Deployment {
	ls := labelsForWordpress(m.Name)

	wordpressContainer := corev1.Container{
		Name:  "wordpress",
		Image: "wordpress",
		Env: []corev1.EnvVar{
			{
				Name:  "WORDPRESS_DB_HOST",
				Value: "localhost:33060",
			},
			{
				Name:  "WORDPRESS_DB_USER",
				Value: "root",
			},
			{
				Name:  "WORDPRESS_DB_PASSWORD",
				Value: "Password1234",
			},
			{
				Name:  "WORDPRESS_DB_NAME",
				Value: "wordpress",
			},
			{
				Name:  "WORDPRESS_TABLE_PREFIX",
				Value: "wp_",
			},
		},
	}

	containers := []corev1.Container{wordpressContainer}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			//Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: containers,
				},
			},
		},
	}
	// Set Wordpress instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// deploymentForWordpress returns a wordpress Deployment object
func (r *WordpressReconciler) deploymentForMysql(m *wordpressfullstackv1.Wordpress) *appsv1.Deployment {
	ls := labelsForMysql(m.Name)

	mysqlContainer := corev1.Container{
		Name:  "mysql",
		Image: "mysql",
		Ports: []corev1.ContainerPort{{
			Name:          "mysql",
			ContainerPort: 3306,
			Protocol:      "TCP",
			HostIP:        "127.0.0.1",
			HostPort:      3306,
		}},
		Env: []corev1.EnvVar{{
			Name:  "MYSQL_ROOT_PASSWORD",
			Value: "Password1234",
		}},
	}
	containers := []corev1.Container{mysqlContainer}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			//Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: containers,
				},
			},
		},
	}

	// Set Wordpress instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForWordpress returns the labels for selecting the resources
// belonging to the given wordpress CR name.
func labelsForDeployment(name string) map[string]string {
	return map[string]string{"app": "wordpress", "wordpress_cr": name}
}

// labelsForWordpress returns the labels for selecting the resources
// belonging to the given wordpress CR name.
func labelsForWordpress(name string) map[string]string {
	return map[string]string{"app": "wordpress", "wordpress_cr": name, "tier": "server"}
}

// labelsForWordpress returns the labels for selecting the resources
// belonging to the given wordpress CR name.
func labelsForMysql(name string) map[string]string {
	return map[string]string{"app": "wordpress", "wordpress_cr": name, "tier": "database"}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

func (r *WordpressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wordpressfullstackv1.Wordpress{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=cache.example.com,resources=wordpresss,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=wordpresss/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

// WordpressReconciler reconciles a Wordpress object
type WordpressReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}
