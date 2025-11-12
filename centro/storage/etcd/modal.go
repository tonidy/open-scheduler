package etcd

type InstanceItem struct {
	JobID        string `json:"job_id"`
	InstanceName string `json:"instance_name"`
	Status       string `json:"status"`
	Created      string `json:"created"`
}
