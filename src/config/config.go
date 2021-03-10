package config

import "path/filepath"

var (
	// NFS_EXPOSED_PATH is the path of the network data which NFS server exposed.
	// This path is needed for Kubernetes deployment volume.
	NFS_EXPOSED_PATH = "/var/mictract"

	// LOCAL_MOUNT_PATH is the path where the network data mounted in each container.
	// This path is needed for Kubernetes deployment volume.
	LOCAL_MOUNT_PATH = "/mictract"

	// LOCAL_BASE_PATH is where the networks folder is actually stored.
	LOCAL_BASE_PATH = filepath.Join(LOCAL_MOUNT_PATH, "networks")

	// The config file path, which to connect k8s.
	K8S_CONFIG = filepath.Join(LOCAL_MOUNT_PATH, "kube-config.yaml")

	NFS_SERVER_URL = "192.168.49.1"

	// LOCAL_BASE_PATH is where the scripts folder is actually stored.
	LOCAL_SCRIPTS_PATH = filepath.Join(LOCAL_MOUNT_PATH, "scripts")

	SDK_LEVEL = "debug"

	MYSQL_PW = "123456"
)
