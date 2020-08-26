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
	"strconv"

	//"reflect"

	wordpressfullstackv1 "github.com/James-Moore/wordpress-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	WordpressAppName       = "wordpress"
	WordpressContainerName = "wordpress"
	WordpressImageName     = "wordpress"

	MySQLContainerName = "mysql"
	MySQLImageName     = "mysql"

	MySQL_MYSQLCNF_ConfigMapKey = "mysqlconfig"
	MySQL_MYSQLCNF_ConfigVolume = "mysqlconfigvolume"
	MySQL_MYSQLCNF_ConfigPath   = "/etc/mysql"
	MySQL_MYSQLCNF_ConfigFile   = "mysql.cnf"

	MySQL_SECURECHK_ConfigMapKey = "mysqlsecurecheck"
	MySQL_SECURECHK_ConfigVolume = "mysqlsecurecheckvolume"
	MySQL_SECURECHK_ConfigPath   = "/var/lib/mysql-files"
	MySQL_SECURECHK_ConfigFile   = "blank.cnf"

	MySQLPort       = "3306"
	MySQLPort_INT32 = 3306

	ServiceName = "wpservice"
)

func (r *WordpressReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	namespaceName := req.NamespacedName
	log := r.Log.WithValues("wordpress", namespaceName)
	ctx := context.Background()

	// Objects to be populated as pass by reference
	wordpress := &wordpressfullstackv1.Wordpress{}
	deployment := &appsv1.Deployment{}
	podList := &corev1.PodList{}

	log.Info("CHECKPOINT 1")
	breakControl, err := r.reconcileCustomResource(ctx, namespaceName, wordpress)
	if breakControl {
		return ctrl.Result{}, err
	}

	log.Info("CHECKPOINT 2")
	breakControl, err = r.reconcileDeployment(ctx, wordpress, deployment)
	if breakControl {
		return ctrl.Result{}, err
	}

	// Update the Wordpress status with the pod names
	log.Info("CHECKPOINT 3")
	breakControl, err = r.getPodList(ctx, wordpress, podList)
	if breakControl {
		return ctrl.Result{}, err
	}

	log.Info("CHECKPOINT 4")
	breakControl, err = r.reconcilePods(ctx, wordpress, podList)
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

	log.Info("CHECKPOINT 5")
	return ctrl.Result{}, nil
}

func (r *WordpressReconciler) getPodState(pod corev1.Pod) string {
	message := "Pod: " + pod.Name
	for _, containerStatus := range pod.Status.ContainerStatuses {
		message = message + ", Container: " + containerStatus.Name + "State: " + containerStatus.State.String()
	}
	return message
}

func (r *WordpressReconciler) getPodsState(pods []corev1.Pod) string {
	message := "Pods: "
	for _, pod := range pods {
		message += r.getPodState(pod)
	}
	return message
}

func (r *WordpressReconciler) containsContainer(containerName string, pod corev1.Pod) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if containerName == cs.Name {
			return true
		}
	}
	return false
}

func (r *WordpressReconciler) isContainerInPodReady(containerName string, pod corev1.Pod) bool {
	ready := false
	for _, cs := range pod.Status.ContainerStatuses {
		if containerName == cs.Name {
			ready = cs.Ready
		}
	}
	return ready
}

//MySQLContainerName
func (r *WordpressReconciler) isContainerInPodsReady(containerName string, podName string, pods []corev1.Pod) bool {
	ready := false

	r.Log.Info(r.getPodsState(pods))
	for _, pod := range pods {
		if (podName == pod.Name) && (r.containsContainer(containerName, pod)) {
			ready = r.isContainerInPodReady(containerName, pod)
			r.Log.Info("MySql Ready: " + strconv.FormatBool(ready))
			return ready
		}
	}

	return ready
}

func (r *WordpressReconciler) reconcilePods(ctx context.Context, wordpress *wordpressfullstackv1.Wordpress, podList *corev1.PodList) (breakControl bool, err error) {
	breakControl = false
	err = nil

	mysqlReady := false

	// TODO Get container name and pod name from the wordpress status to feed into isContainerPodsReady
	//mysqlReady := r.isContainerInPodsReady()

	//Nothing to reconcile if mysql is still booting in the container
	if !mysqlReady {
		breakControl = true
		return breakControl, err
	}

	// TODO Add wordpress to deployment here
	//, r.defineWordpressContainer(wordpress)

	//podNames := getPodNames(podList.Items)
	//Development statement:  Printing all pod names
	//r.Log.Info("PRINTING POD NAMES: ")
	//for podName, _ := range podNames {
	//	r.Log.Info(podName)
	//}

	//switch os := len(podNames); os {
	//case 0:
	//	err := errors2.New("wordpress pod reconciliation error")
	//	r.Log.Error(err, "WordpressContainer and MySQLContainer do not exist in deployment.  This should never happen.", "Wordpress.Namespace", wordpress.Namespace, "Wordpress.Name", wordpress.Name)
	//	return err
	//case 1:
	//	if _, ok := podNames[WordpressImageName]; !ok {
	//		err := errors2.New("wordpress pod reconciliation error")
	//		r.Log.Error(err, "WordpressContainer exists without MySQLContainer.  This should never happen.", "Wordpress.Namespace", wordpress.Namespace, "Wordpress.Name", wordpress.Name)
	//		return err
	//	}
	//	// TODO Update Deployment Definition here to include the wordpress container in the Pod.
	//	// This becasue the mysqlpod is up and running
	//default:
	//	if os > 2 {
	//		err := errors2.New("wordpress pod reconciliation error")
	//		num := strconv.Itoa(os)
	//		r.Log.Error(err, "There are additional pods that shoudl not exist in the Wordpress Deployment.  Number of pods are: "+num+".This should never happen.", "Wordpress.Namespace", wordpress.Namespace, "Wordpress.Name", wordpress.Name)
	//		return err
	//	}
	//}

	//Everything went fine.
	return false, nil
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

func (r *WordpressReconciler) getPodList(ctx context.Context, wordpress *wordpressfullstackv1.Wordpress, podList *corev1.PodList) (breakControl bool, err error) {
	breakControl = false
	err = nil

	// Define the values to match Pods against
	listOpts := []client.ListOption{
		client.InNamespace(wordpress.Namespace),
		client.MatchingLabels(labelsForDeployment(wordpress.Name)),
	}

	// Get the list of pods matching our search parameters: Namespace and Labels
	err = r.List(ctx, podList, listOpts...)
	if err != nil {
		//An error occurred while retrieving the pods in our deployment.  Break control loop and return with error.
		breakControl = true
		r.Log.Error(err, "Failed to list pods", "Wordpress.Namespace", wordpress.Namespace, "Wordpress.Name", wordpress.Name)
		return breakControl, err
	}

	//The deployment pods have been retreived successfully.  Continue control loop.
	return breakControl, err
}

func (r *WordpressReconciler) reconcileCustomResource(ctx context.Context, namespaceName types.NamespacedName, wordpress *wordpressfullstackv1.Wordpress) (breakControl bool, err error) {
	breakControl = false
	err = nil

	// Fetch the Wordpress instance
	err = r.Get(ctx, namespaceName, wordpress)
	if err != nil {
		breakControl = true

		if errors.IsNotFound(err) {
			// Request custom resource not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info("Wordpress resource not found. Ignoring since object must be deleted")
			err = nil
			return breakControl, err
		} else {
			//The custom resource may exist.  We dont know in this case because an error was thrown while
			//attempting cr retrieval.  Therefore we have to break control loop and exit.
			r.Log.Error(err, "Failed to get Wordpress")
			return breakControl, err
		}
	}

	//Wordpress CR was retrieved successfully so do not break control loop.
	return breakControl, err
}

func (r *WordpressReconciler) reconcileDeployment(ctx context.Context, wordpress *wordpressfullstackv1.Wordpress, deployment *appsv1.Deployment) (breakControl bool, err error) {
	breakControl = false
	err = nil

	// Check if the deployment already exists, if not create a new one
	err = r.Get(ctx, types.NamespacedName{Name: wordpress.Name, Namespace: wordpress.Namespace}, deployment)
	if err != nil {

		//If the deployment is not found create the deployment and disregard the error.  This is expected behavior.
		if errors.IsNotFound(err) {
			// Create the containers
			r.Log.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)

			// Define a initial container in deployment.  We start with MySql
			containers := []corev1.Container{r.defineWordpressContainer(wordpress), r.defineMySqlContainer(wordpress)}

			// Define the deployment
			deployment := r.defineDeployment(wordpress, containers)

			// Set Wordpress instance as the owner and controller
			ctrl.SetControllerReference(wordpress, deployment, r.Scheme)

			err = r.Create(ctx, deployment)
			if err != nil {
				breakControl = true
				r.Log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
				return breakControl, err
			}

			//TODO potentially assign Wordpress CR UID to Deployment as a label to allow searching for only this object
			//Assign the wordpress deployment data to the status
			wordpress.Status.WordpressDeploymentName = deployment.Name
			wordpress.Status.WordpressDeploymentNamespace = deployment.Namespace
			wordpress.Status.WordpressDeploymentUUID = string(deployment.UID)

			//Assign the Wordpress pod data to the CR status
			wordpress.Status.WordpressPodName = deployment.Spec.Template.Name
			wordpress.Status.WordpressPodNamespace = deployment.Spec.Template.Namespace
			wordpress.Status.WordpressPodUID = string(deployment.Spec.Template.UID)

			//Assign the mysql container name to the cr status
			//Note: We can index zero here only because we created the array and we know there is only 1 container
			wordpress.Status.WordpressContainerName = deployment.Spec.Template.Spec.Containers[0].Name

			//Must update wordpress cr to include added status values
			err := r.Status().Update(ctx, wordpress)
			if err != nil {
				breakControl = true
				r.Log.Error(err, "Failed to update Wordpress status")
				return breakControl, err
			}

			// Deployment created successfully - disregard err - return and requeue
			err = nil
			return breakControl, err

		} else {
			//If the error is the result of anyting but not finding the deployment then
			//an error was encountered when trying to get the deployment.  So exit control loop.
			breakControl = true
			r.Log.Error(err, "Failed to get Deployment")
			return breakControl, err
		}

	}

	//If we have arrived here then we have not encountered an error
	//We have succesfully retrieved the deployment.  So continue control loop.
	return breakControl, err
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
					Volumes: []corev1.Volume{{
						Name: MySQL_MYSQLCNF_ConfigVolume,
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: MySQL_MYSQLCNF_ConfigMapKey},
								Items: []corev1.KeyToPath{{
									Key:  MySQL_MYSQLCNF_ConfigFile,
									Path: MySQL_MYSQLCNF_ConfigFile,
								}},
							},
						},
					},
						{
							Name: MySQL_SECURECHK_ConfigVolume,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: MySQL_SECURECHK_ConfigMapKey},
									Items: []corev1.KeyToPath{{
										Key:  MySQL_SECURECHK_ConfigFile,
										Path: MySQL_SECURECHK_ConfigFile,
									}},
								},
							},
						}},
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
		Ports: []corev1.ContainerPort{{
			Name:          "wordpress",
			ContainerPort: 80,
		}},
		Env: []corev1.EnvVar{
			{
				Name:  "WORDPRESS_DB_HOST",
				Value: ServiceName + ":" + MySQLPort,
			},
			{
				Name:  "WORDPRESS_DB_USER",
				Value: "root",
			},
			{
				Name:  "WORDPRESS_DB_PASSWORD",
				Value: m.Spec.Password,
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

	probeAction := &corev1.ExecAction{
		Command: []string{"sh", "-c", "mysqladmin ping -u root -p${MYSQL_ROOT_PASSWORD}"},
	}

	probeHandler := corev1.Handler{
		Exec: probeAction,
	}

	readinessProbe := &corev1.Probe{
		Handler:             probeHandler,
		InitialDelaySeconds: 10,
		PeriodSeconds:       5,
	}

	mysqlContainer := corev1.Container{
		Name:  MySQLContainerName,
		Image: MySQLImageName,
		Ports: []corev1.ContainerPort{{
			Name:          "mysql",
			ContainerPort: MySQLPort_INT32,
		}},
		Env: []corev1.EnvVar{{
			Name:  "MYSQL_ROOT_PASSWORD",
			Value: m.Spec.Password,
		}},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      MySQL_MYSQLCNF_ConfigVolume,
			MountPath: MySQL_MYSQLCNF_ConfigPath,
		},
			{
				Name:      MySQL_SECURECHK_ConfigVolume,
				MountPath: MySQL_SECURECHK_ConfigPath,
			},
		},
		ReadinessProbe: readinessProbe,
	}

	return mysqlContainer
}

// labelsForWordpress returns the labels for selecting the resources
// belonging to the given wordpress CR name.
func labelsForDeployment(name string) map[string]string {
	return map[string]string{"app": WordpressAppName, "wordpress_cr": name}
}

func (r *WordpressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr)
	builder = builder.For(&wordpressfullstackv1.Wordpress{})
	builder = builder.Owns(&appsv1.Deployment{})
	return builder.Complete(r)

	//return ctrl.NewControllerManagedBy(mgr).
	//	For(&wordpressfullstackv1.Wordpress{}).
	//	Owns(&appsv1.Deployment{}).
	//	Complete(r)

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
