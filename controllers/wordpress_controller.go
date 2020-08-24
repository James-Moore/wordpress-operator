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
	errors2 "errors"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	//"reflect"

	wordpressfullstackv1 "github.com/James-Moore/wordpress-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	WordpressAppName       = "wordpress"
	WordpressContainerName = "wordpress"
	WordpressImageName     = "wordpress"
	MySQLContainerName     = "mysql"
	MySQLImageName         = "mysql"
)

func (r *WordpressReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	namespaceName := req.NamespacedName
	log := r.Log.WithValues("wordpress", namespaceName)
	ctx := context.Background()
	wordpress := &wordpressfullstackv1.Wordpress{}
	deployment := &appsv1.Deployment{}
	podList := &corev1.PodList{}

	log.Info("CHECKPOINT 1")
	err := r.ReconcileCustomResource(ctx, wordpress, namespaceName)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("CHECKPOINT 2")
	err = r.ReconcileDeployment(ctx, wordpress, deployment)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Update the Wordpress status with the pod names
	log.Info("CHECKPOINT 3")
	err = r.getPodList(ctx, wordpress, podList)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("CHECKPOINT 4")
	err = r.ReconcilePods(ctx, wordpress, podList)
	if err != nil {
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

	log.Info("CHECKPOINT 5")
	return ctrl.Result{}, nil
}

func (r *WordpressReconciler) ReconcilePods(ctx context.Context, wordpress *wordpressfullstackv1.Wordpress, podList *corev1.PodList) error {
	podNames := getPodNames(podList.Items)
	switch os := len(podNames); os {
	case 0:
		err := errors2.New("Wordpress Pod Reconciliation Error")
		r.Log.Error(err, "WordpressContainer and MySQLContainer do not exist in deployment.  This should never happen.", "Wordpress.Namespace", wordpress.Namespace, "Wordpress.Name", wordpress.Name)
		return err
	case 1:
		if _, ok := podNames[WordpressImageName]; !ok {
			err := errors2.New("Wordpress Pod Reconciliation Error")
			r.Log.Error(err, "WordpressContainer exists without MySQLContainer.  This should never happen.", "Wordpress.Namespace", wordpress.Namespace, "Wordpress.Name", wordpress.Name)
			return err
		}
		// TODO Update Deployment Definition here to include the wordpress container in the Pod.
		// This becasue the mysqlpod is up and running
	default:
		if os > 2 {
			err := errors2.New("Wordpress Pod Reconciliation Error")
			r.Log.Error(err, "There are additional pods that shoudl not exist in the Wordpress Deployment.  Number of pods are: "+string(os)+".This should never happen.", "Wordpress.Namespace", wordpress.Namespace, "Wordpress.Name", wordpress.Name)
			return err
		}
	}

	//Development statement:  Printing all pod names
	for podName := range podNames {
		r.Log.Info(podName)
	}

	//Everything went fine.
	return nil
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) map[string]bool {
	podNames := make(map[string]bool) // New empty set
	//delete(podNames, "Foo")    // Delete
	//exists := podNames["Foo"]  // Membership

	for _, pod := range pods {
		podNames[pod.Name] = true // Add
	}
	return podNames
}

func (r *WordpressReconciler) getPodList(ctx context.Context, wordpress *wordpressfullstackv1.Wordpress, podList *corev1.PodList) error {
	// Define the values to match Pods against
	listOpts := []client.ListOption{
		client.InNamespace(wordpress.Namespace),
		client.MatchingLabels(labelsForDeployment(wordpress.Name)),
	}

	// Get the list of pods matching our search parameters: Namespace and Labels
	err := r.List(ctx, podList, listOpts...)
	if err != nil {
		r.Log.Error(err, "Failed to list pods", "Wordpress.Namespace", wordpress.Namespace, "Wordpress.Name", wordpress.Name)
		return err
	}

	return nil
}

func (r *WordpressReconciler) ReconcileCustomResource(ctx context.Context, wordpress *wordpressfullstackv1.Wordpress, namespaceName types.NamespacedName) error {
	// Fetch the Wordpress instance
	err := r.Get(ctx, namespaceName, wordpress)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info("Wordpress resource not found. Ignoring since object must be deleted")
			return nil
		}
		// Error reading the object - requeue the request.
		r.Log.Error(err, "Failed to get Wordpress")
		return err
	}

	return nil
}

func (r *WordpressReconciler) ReconcileDeployment(ctx context.Context, wordpress *wordpressfullstackv1.Wordpress, deployment *appsv1.Deployment) error {
	// Check if the deployment already exists, if not create a new one
	err := r.Get(ctx, types.NamespacedName{Name: wordpress.Name, Namespace: wordpress.Namespace}, deployment)

	//If
	if err != nil && errors.IsNotFound(err) {
		// Create the containers
		r.Log.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)

		// Define a initial container in deployment.  We start with MySql
		containers := []corev1.Container{r.defineMySqlContainer(wordpress), r.defineWordpressContainer(wordpress)}

		// Define the deployment
		deployment := r.defineDeployment(wordpress, containers)

		// Set Wordpress instance as the owner and controller
		ctrl.SetControllerReference(wordpress, deployment, r.Scheme)

		err = r.Create(ctx, deployment)

		if err != nil {
			r.Log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return err
		}

		// Deployment created successfully - return and requeue
		return nil

	} else if err != nil {
		r.Log.Error(err, "Failed to get Deployment")
		return err
	}

	return nil
}

// deploymentForWordpress returns a wordpress Deployment object
func (r *WordpressReconciler) defineDeployment(w *wordpressfullstackv1.Wordpress, containers []corev1.Container) *appsv1.Deployment {
	ls := labelsForDeployment(w.Name)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      w.Name,
			Namespace: w.Namespace,
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

	return dep
}

// deploymentForWordpress returns a wordpress Deployment object
func updateDeploymentDefinition(deployment *appsv1.Deployment, container corev1.Container) {
	containers := deployment.Spec.Template.Spec.Containers
	deployment.Spec.Template.Spec.Containers = append(containers, container)
}

func (r *WordpressReconciler) defineWordpressContainer(m *wordpressfullstackv1.Wordpress) corev1.Container {
	wordpressContainer := corev1.Container{
		Name:  WordpressContainerName,
		Image: WordpressImageName,
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

func (r *WordpressReconciler) defineMySqlContainer(m *wordpressfullstackv1.Wordpress) corev1.Container {

	mysqlContainer := corev1.Container{
		Name:  MySQLContainerName,
		Image: MySQLImageName,
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

// labelsForWordpress returns the labels for selecting the resources
// belonging to the given wordpress CR name.
func labelsForDeployment(name string) map[string]string {
	return map[string]string{"app": WordpressAppName, "wordpress_cr": name}
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
	Log           logr.Logger
	Scheme        *runtime.Scheme
	WordpressName string
	MysqlName     string
}
