package pod

import (
	"github.com/James-Moore/wordpress-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strconv"
	"time"
)

type PodManager struct {
	Common *common.Common
}

func (manager *PodManager) getPodState(pod corev1.Pod) string {
	message := "Pod: " + pod.Name
	for _, containerStatus := range pod.Status.ContainerStatuses {
		message = message + ", Container: " + containerStatus.Name + "State: " + containerStatus.State.String()
	}
	return message
}

func (manager *PodManager) getPodsState(pods []corev1.Pod) string {
	message := "Pods: "
	for _, pod := range pods {
		message += manager.getPodState(pod)
	}
	return message
}

func (manager *PodManager) containsContainer(containerName string, pod corev1.Pod) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if containerName == cs.Name {
			return true
		}
	}
	return false
}

func (manager *PodManager) isContainerInPodReady(containerName string, pod corev1.Pod) bool {
	ready := false
	for _, cs := range pod.Status.ContainerStatuses {
		if containerName == cs.Name {
			ready = cs.Ready
		}
	}
	return ready
}

//MYSQL_CONTAINER_NAME
func (manager *PodManager) isContainerInPodsReady(containerName string, pods []corev1.Pod) bool {
	ready := false

	manager.Common.Log.Info(manager.getPodsState(pods))
	manager.Common.Log.Info("Looking for ContainerNamed " + containerName)
	for _, pod := range pods {
		if manager.containsContainer(containerName, pod) {
			manager.Common.Log.Info("ContainerName match found..")
			ready = manager.isContainerInPodReady(containerName, pod)
			manager.Common.Log.Info("MySql Ready: " + strconv.FormatBool(ready))
			return ready
		}
	}

	return ready
}

func (manager *PodManager) Reconcile() (result reconcile.Result, breakControl bool, err error) {
	result = reconcile.Result{}
	breakControl = false
	err = nil

	podList := &corev1.PodList{}
	breakControl, err = manager.getPodList(podList)
	if err != nil {
		manager.Common.Log.Error(err, "Could not retrieve pods from deployment while Reconciling.")
		breakControl = true
		return result, breakControl, err
	}

	// TODO Get container name and pod name from the wordpress status to feed into isContainerPodsReady
	mysqlContainerName := manager.Common.Wordpress.Status.MySqlContainerName
	mysqlReady := manager.isContainerInPodsReady(mysqlContainerName, podList.Items)

	//Nothing to reconcile if mysql is still booting in the container
	if !mysqlReady {
		result = reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}
		manager.Common.Log.Info("MySQL is ready: " + strconv.FormatBool(mysqlReady))
		breakControl = true
		return result, breakControl, err
	}

	manager.Common.Log.Info("MySQL is ready: " + strconv.FormatBool(mysqlReady))

	//Everything went fine.
	return result, false, nil
}

// getPodNames returns the pod names of the array of pods passed in
func (manager *PodManager) getPodNames(pods []corev1.Pod) map[string]bool {
	podNames := make(map[string]bool) // New empty set
	//delete(podNames, "Foo")    // Delete
	//exists := podNames["Foo"]  // Membership

	for _, pod := range pods {
		podNames[pod.Name] = true // Add
	}
	return podNames
}

func (manager *PodManager) getPodList(podList *corev1.PodList) (breakControl bool, err error) {
	breakControl = false
	err = nil

	// Define the values to match Pods against
	listOpts := []client.ListOption{
		client.InNamespace(manager.Common.Wordpress.Namespace),
		client.MatchingLabels(manager.Common.GetSelectionLabels(common.CURRENT_APPLICATION_TIERING)),
	}

	// Get the list of pods matching our search parameters: Namespace and Labels
	err = manager.Common.Client.List(manager.Common.Ctx, podList, listOpts...)
	if err != nil {
		//An error occurred while retrieving the pods in our deployment.  Break control loop and return with error.
		breakControl = true
		manager.Common.Log.Error(err, "Failed to list pods", "Wordpress.Namespace", manager.Common.GetNamespace(), "Wordpress.Name", manager.Common.GetCustomResourceValue())
		return breakControl, err
	}

	//The deployment pods have been retreived successfully.  Continue control loop.
	return breakControl, err
}
