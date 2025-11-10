package model

type ContainerStatus string

const (
	ContainerStatusRunning ContainerStatus = "running"
	ContainerStatusStopped ContainerStatus = "stopped"
	ContainerStatusExited  ContainerStatus = "exited"
	ContainerStatusFailed  ContainerStatus = "failed"
)

// ContainerInspect reflects the output of `podman container inspect` (summary)
type ContainerInspect struct {
	ID         string            `json:"Id"`
	Name       string            `json:"Name"`
	Image      string            `json:"Image"`
	ImageName  string            `json:"ImageName"`
	Command    []string          `json:"Command"`
	Args       []string          `json:"Args"`
	Created    string            `json:"Created"`
	StartedAt  string            `json:"StartedAt"`
	FinishedAt string            `json:"FinishedAt"`
	Status     ContainerStatus   `json:"Status"`
	ExitCode   int               `json:"ExitCode"`
	Pid        int               `json:"Pid"`
	Labels     map[string]string `json:"Labels,omitempty"`
	Ports      []string          `json:"Ports,omitempty"`
	Volumes    []string          `json:"Volumes,omitempty"`
}