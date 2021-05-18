package config

import (
	"os"
	"path/filepath"
)

var (
	// NFS_EXPOSED_PATH is the path of the network data which NFS server exposed.
	// This path is needed for Kubernetes deployment volume.
	NFS_EXPOSED_PATH	= "/var/mictract"

	// LOCAL_MOUNT_PATH is the path where the network data mounted in each container.
	// This path is needed for Kubernetes deployment volume.
	LOCAL_MOUNT_PATH	= "/mictract"

	// LOCAL_BASE_PATH is where the networks folder is actually stored.
	LOCAL_BASE_PATH		= filepath.Join(LOCAL_MOUNT_PATH, "networks")

	// The config file path, which to connect k8s.
	K8S_CONFIG			= filepath.Join(LOCAL_MOUNT_PATH, "kube-config.yaml")

	// LOCAL_BASE_PATH is where the scripts folder is actually stored.
	LOCAL_SCRIPTS_PATH	= filepath.Join(LOCAL_MOUNT_PATH, "scripts")

	LOCAL_CC_PATH		= filepath.Join(LOCAL_MOUNT_PATH, "chaincodes")

	SDK_LEVEL			= "info"

	// Do not modify the super user id
	Super_User_ID		= -1
	Super_User_PW		= "root"

	// export A_B_C = D_E_F
	NFS_SERVER_URL		= os.Getenv("NFS_SERVER_URL")
	DB_SERVER_URL		= os.Getenv("DB_SERVER_URL")
	DB_PW         		= os.Getenv("DB_PW")

	// alert
	// example: export ALERT_ENABLE = true
	ALERT_ENABLE		= os.Getenv("ALERT_ENABLE")

	ES_HOSTS			= []string{"http://elasticsearch-logging.kube-system:9200/"}
	SMTPHost			= os.Getenv("SMTPHost")
	SMTPPort			= os.Getenv("SMTPPort")
	SMTPUsername		= os.Getenv("SMTPUsername")
	SMTPPassword		= os.Getenv("SMTPPassword")
	// example: export SMTPRecvier=zhangsan@qq.com;lisi@qq.com
	SMTPRecvier			= os.Getenv("SMTPRecvier")
)
