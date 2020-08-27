package common

import (
	"context"
	wordpressfullstackv1 "github.com/James-Moore/wordpress-operator/api/v1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=wordpress-fullstack.jamesmoore.in,resources=wordpresss,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wordpress-fullstack.jamesmoore.in,resources=wordpresss/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
type Common struct {
	Client    client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Ctx       context.Context
	Wordpress *wordpressfullstackv1.Wordpress
}

func (common *Common) Init(namespaceName types.NamespacedName) (breakControl bool, err error) {
	breakControl = false
	err = nil

	err = common.Client.Get(common.Ctx, namespaceName, common.Wordpress)
	if err != nil {
		breakControl = true

		if errors.IsNotFound(err) {
			// Request custom resource not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			common.Log.Info("Wordpress resource not found. Ignoring since object must be deleted")
			err = nil
			return breakControl, err
		} else {
			//The custom resource may exist.  We dont know in this case because an error was thrown while
			//attempting cr retrieval.  Therefore we have to break control loop and exit.
			common.Log.Error(err, "Failed to get Wordpress")
			return breakControl, err
		}
	}

	//Wordpress CR was retrieved successfully so do not break control loop.
	return breakControl, err
}

func (common *Common) GetNamespace() string {
	return common.Wordpress.Namespace
}
func (common *Common) GetApplicationName() string {
	return common.Wordpress.Spec.AppName
}
func (common *Common) GetCustomResourceType() string {
	return common.Wordpress.Kind
}
func (common *Common) GetCustomResourceValue() string {
	return common.Wordpress.Name
}
func (common *Common) GetDeployment() (*v1.Deployment, error) {
	deployment := &v1.Deployment{}
	err := common.Client.Get(common.Ctx, common.GetNamespacedName(), deployment)
	return deployment, err
}

// Returns the labels for selecting the resources
func (common *Common) GetSelectionLabels(tier string) map[string]string {
	//Example: map[string]string{"app": "mywordpressapp", "wordpress_cr": "mywordpress"}
	return map[string]string{APPLICATION_KEY: common.GetApplicationName(), common.GetCustomResourceType(): common.GetCustomResourceValue(), TIER_KEY: tier}
}

// Returns the NamespacedName for selecting the resources
func (common *Common) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{Namespace: common.GetNamespace(), Name: common.GetCustomResourceValue()}
}
