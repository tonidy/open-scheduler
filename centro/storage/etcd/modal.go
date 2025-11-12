package etcd

type ContainerItem struct {
	ContainerName string `json:"container_name"`	
	Status        string `json:"status"`
	Created       string `json:"created"`
}