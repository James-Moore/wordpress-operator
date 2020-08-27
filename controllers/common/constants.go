package common

const (
	CURRENT_APPLICATION_TIERING = TIER_FULLSTACK

	WORDPRESS_APP_NAME       = "wordpress_on_sql"
	WORDPRESS_CONTAINER_NAME = "wordpress"
	WORDPRESS_IMAGE_NAME     = "wordpress"
	MYSQL_CONTAINER_NAME     = "mysql"
	MYSQL_IMAGE_NAME         = "mysql"

	MYSQL_MYSQLCNF_CONFIGMAP_KEY    = "mysqlconfig"
	MySQL_MYSQLCNF_CONFIGMAP_VOLUME = "mysqlconfigvolume"
	MYSQL_MYSQLCNF_CONFIGMAP_PATH   = "/etc/mysql"
	MYSQL_MYSQLCNF_CONFIG_FILE      = "mysql.cnf"

	MYSQL_SECURECHK_CONFIGMAP_KEY    = "mysqlsecurecheck"
	MYSQL_SECURECHK_CONFIGMAP_VOLUME = "mysqlsecurecheckvolume"
	MYSQL_SECURECHK_CONFIGMAP_PATH   = "/var/lib/mysql-files"
	MYSQL_SECURECHK_CONFIGMAP_FILE   = "blank.cnf"

	MYSQL_PORT_STRING = "3306"
	MYSQL_PORT_INT32  = 3306

	SERVICE_NAME = "wpservice"
)

const (
	APPLICATION_KEY = "app"
	TIER_KEY        = "tier"
	TIER_FRONTEND   = "front"
	TIER_BACKEND    = "back"
	TIER_FULLSTACK  = "fullstack"
)
