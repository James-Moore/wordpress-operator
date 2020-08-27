package configmap

import (
	"github.com/James-Moore/wordpress-operator/controllers/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

type ConfigMapManager struct {
	C *common.Common
}

type ConfigMapFileType struct {
	Namespace string
	Mapkey    string
	Directory string
	File      string
	ConfigMap *corev1.ConfigMap
}

func (manager *ConfigMapManager) Reconcile() (breakcontrol bool, err error) {
	breakcontrol = false
	err = nil

	mysqlcnfConfigMap := ConfigMapFileType{
		Namespace: "default",
		Mapkey:    common.MYSQL_MYSQLCNF_CONFIGMAP_KEY,
		Directory: common.MYSQL_MYSQLCNF_CONFIGMAP_PATH,
		File:      common.MYSQL_MYSQLCNF_CONFIGMAP_FILE,
	}
	err = mysqlcnfConfigMap.PopulateConfigmap()
	if err != nil {
		breakcontrol = true
		return breakcontrol, err
	}

	// Create the containers
	manager.C.Log.Info("Creating a new Configmap", "Namespace", mysqlcnfConfigMap.Namespace, "Name", mysqlcnfConfigMap.Mapkey)
	err = manager.C.Client.Create(manager.C.Ctx, mysqlcnfConfigMap.ConfigMap)
	if err != nil {
		breakcontrol = true
		manager.C.Log.Error(err, "Failed to create new ConfigMap", "Namespace", mysqlcnfConfigMap.Namespace, "Name", mysqlcnfConfigMap.Mapkey)
		return breakcontrol, err
	}
	manager.C.Log.Info("Updating Custom Resource Status")
	manager.C.Wordpress.Status.MySqlConfigMap_Configuration = string(mysqlcnfConfigMap.ConfigMap.UID)
	manager.C.Log.Info("MySql Configuration ConfigMap Added To Wordpress CR Status")

	securechkConfigMap := ConfigMapFileType{
		Namespace: "default",
		Mapkey:    common.MYSQL_SECURECHK_CONFIGMAP_KEY,
		Directory: common.MYSQL_SECURECHK_CONFIGMAP_PATH,
		File:      common.MYSQL_SECURECHK_CONFIGMAP_FILE,
	}
	err = securechkConfigMap.PopulateConfigmap()
	if err != nil {
		breakcontrol = true
		return breakcontrol, err
	}

	// Create the containers
	manager.C.Log.Info("Creating a new Configmap", "Namespace", securechkConfigMap.Namespace, "Name", securechkConfigMap.Mapkey)
	err = manager.C.Client.Create(manager.C.Ctx, securechkConfigMap.ConfigMap)
	if err != nil {
		breakcontrol = true
		manager.C.Log.Error(err, "Failed to create new ConfigMap", "Namespace", mysqlcnfConfigMap.Namespace, "Name", mysqlcnfConfigMap.Mapkey)
		return breakcontrol, err
	}
	manager.C.Log.Info("Updating Custom Resource Status")
	manager.C.Wordpress.Status.MySqlConfigmap_Secure = string(securechkConfigMap.ConfigMap.UID)
	manager.C.Log.Info("MySql Secure ConfigMap Added To Wordpress CR Status")

	return breakcontrol, err
}

func (cmft *ConfigMapFileType) PopulateConfigmap() error {
	config := Config{}

	err := config.LoadConfig(cmft.Directory + string(os.PathSeparator) + cmft.File)
	if err != nil {
		return err
	}

	cmft.ConfigMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmft.Mapkey,
			Namespace: cmft.Namespace,
		},
		Data: map[string]string{
			cmft.File: config.Message,
		},
	}

	return err
}
