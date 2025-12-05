package etcd

type InstanceItem struct {
	DeploymentID string `json:"deployment_id"`
	InstanceName string `json:"instance_name"`
	Status       string `json:"status"`
	Created      string `json:"created"`
}
