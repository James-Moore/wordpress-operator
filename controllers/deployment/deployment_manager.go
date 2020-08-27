package deployment

import (
	wordpressfullstackv1 "github.com/James-Moore/wordpress-operator/api/v1"
	"github.com/James-Moore/wordpress-operator/controllers/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type DeploymentManager struct {
	C *common.Common
}

// deploymentForWordpress returns a Wordpress Deployment object
func (manager *DeploymentManager) populateDeployment(containers []corev1.Container) (deployment *appsv1.Deployment) {
	deployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      manager.C.Wordpress.Name,
			Namespace: manager.C.Wordpress.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			//Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: manager.C.GetSelectionLabels(common.CURRENT_APPLICATION_TIERING),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: manager.C.GetSelectionLabels(common.CURRENT_APPLICATION_TIERING),
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name: common.MySQL_MYSQLCNF_CONFIGMAP_VOLUME,
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: common.MYSQL_MYSQLCNF_CONFIGMAP_KEY},
								Items: []corev1.KeyToPath{{
									Key:  common.MYSQL_MYSQLCNF_CONFIG_FILE,
									Path: common.MYSQL_MYSQLCNF_CONFIG_FILE,
								}},
							},
						},
					},
						{
							Name: common.MYSQL_SECURECHK_CONFIGMAP_VOLUME,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: common.MYSQL_SECURECHK_CONFIGMAP_KEY},
									Items: []corev1.KeyToPath{{
										Key:  common.MYSQL_SECURECHK_CONFIGMAP_FILE,
										Path: common.MYSQL_SECURECHK_CONFIGMAP_FILE,
									}},
								},
							},
						}},
					Containers: containers,
				},
			},
		},
	}
	return deployment
}

func (manager *DeploymentManager) updateStatus(deployment *appsv1.Deployment, wordpressContainer *corev1.Container, mysqlContainer *corev1.Container) {
	//TODO potentially assign Wordpress CR UID to Deployment as a label to allow searching for only this object
	//Assign the Wordpress Deployment data to the status
	manager.C.Wordpress.Status.WordpressDeploymentName = deployment.Name
	manager.C.Wordpress.Status.WordpressDeploymentNamespace = deployment.Namespace
	manager.C.Wordpress.Status.WordpressDeploymentUUID = string(deployment.UID)

	//Assign the Wordpress pod data to the CR status
	manager.C.Wordpress.Status.WordpressPodName = deployment.Spec.Template.Name
	manager.C.Wordpress.Status.WordpressPodNamespace = deployment.Spec.Template.Namespace
	manager.C.Wordpress.Status.WordpressPodUID = string(deployment.Spec.Template.UID)

	//Assign the mysql container name to the cr status
	//Note: We can index zero here only because we created the array and we know there is only 1 container
	manager.C.Wordpress.Status.WordpressContainerName = wordpressContainer.Name
	manager.C.Wordpress.Status.MySqlContainerName = mysqlContainer.Name
}

func (manager *DeploymentManager) Reconcile() (bool, error) {
	breakControl := false

	// Check if the Deployment already exists, if not create a new one
	deployment, err := manager.C.GetDeployment()
	if err != nil {
		manager.C.Log.Info("MADE IT HERE AT LEAST ONCE")

		//If the Deployment is not found create the Deployment and disregard the error.  This is expected behavior.
		if errors.IsNotFound(err) {

			// DEBUG

			// Define containers to be deployed
			wordpressContainer := manager.DefineWordpressContainer(manager.C.Wordpress)
			mysqlContainer := manager.DefineMySqlContainer(manager.C.Wordpress)
			containers := []corev1.Container{wordpressContainer, mysqlContainer}

			// Populate the Deployment object
			deployment = manager.populateDeployment(containers)

			// Set Wordpress instance as the owner and controller
			_ = ctrl.SetControllerReference(manager.C.Wordpress, deployment, manager.C.Scheme)

			// Create the containers
			manager.C.Log.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			err = manager.C.Client.Create(manager.C.Ctx, deployment)
			if err != nil {
				breakControl = true
				manager.C.Log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
				return breakControl, err
			}

			manager.updateStatus(deployment, &wordpressContainer, &mysqlContainer)

			//Must update Wordpress cr to include added status values
			err := manager.C.Client.Status().Update(manager.C.Ctx, manager.C.Wordpress)
			if err != nil {
				breakControl = true
				manager.C.Log.Error(err, "Failed to update Wordpress status")
				return breakControl, err
			}

			// Deployment created successfully - disregard err - return and requeue
			err = nil
			return breakControl, err

		} else {
			//If the error is the result of anyting but not finding the Deployment then
			//an error was encountered when trying to get the Deployment.  So exit control loop.
			breakControl = true
			//manager.C.Log.Error(err, "Failed to get Deployment")
			return breakControl, err
		}

	}

	//If we have arrived here then we have not encountered an error
	//We have succesfully retrieved the Deployment.  So continue control loop.
	return breakControl, err
}

func (manager *DeploymentManager) DefineWordpressContainer(m *wordpressfullstackv1.Wordpress) corev1.Container {

	wordpressContainer := corev1.Container{
		Name:  common.WORDPRESS_CONTAINER_NAME,
		Image: common.WORDPRESS_IMAGE_NAME,
		Ports: []corev1.ContainerPort{{
			Name:          "wordpress",
			ContainerPort: 80,
		}},
		Env: []corev1.EnvVar{
			{
				Name:  "WORDPRESS_DB_HOST",
				Value: common.SERVICE_NAME + ":" + common.MYSQL_PORT_STRING,
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

func (manager *DeploymentManager) DefineMySqlContainer(m *wordpressfullstackv1.Wordpress) corev1.Container {

	probeAction := &corev1.ExecAction{
		Command: []string{"mysqladmin", "ping", "-u", "root", "-p${MYSQL_ROOT_PASSWORD}"},
	}

	probeHandler := corev1.Handler{
		Exec: probeAction,
	}

	readinessProbe := &corev1.Probe{
		Handler:             probeHandler,
		InitialDelaySeconds: 20,
		PeriodSeconds:       10,
	}

	mysqlContainer := corev1.Container{
		Name:  common.MYSQL_CONTAINER_NAME,
		Image: common.MYSQL_IMAGE_NAME,
		Ports: []corev1.ContainerPort{{
			Name:          "mysql",
			ContainerPort: common.MYSQL_PORT_INT32,
		}},
		Env: []corev1.EnvVar{{
			Name:  "MYSQL_ROOT_PASSWORD",
			Value: m.Spec.Password,
		}},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      common.MySQL_MYSQLCNF_CONFIGMAP_VOLUME,
			MountPath: common.MYSQL_MYSQLCNF_CONFIGMAP_PATH,
		},
			{
				Name:      common.MYSQL_SECURECHK_CONFIGMAP_VOLUME,
				MountPath: common.MYSQL_SECURECHK_CONFIGMAP_PATH,
			},
		},
		ReadinessProbe: readinessProbe,
	}

	return mysqlContainer
}
