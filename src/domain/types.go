package domain

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
	"nhooyr.io/websocket"
)

const (
	// Used internally (by the frontend) to get the current kubernetes nodes from the backend.
	KUBERNETES_NODES_ENDPOINT = "/api/node"

	// Used internally (by the frontend) to get the system config from the backend.
	SYSTEM_CONFIG_ENDPOINT = "/api/config"

	// Used internally (by the frontend) to get the current set of Jupyter kernel specs from the backend.
	KERNEL_SPEC_ENDPOINT = "/api/kernelspec"
)

var (
	KernelStatuses      = []string{"unknown", "starting", "idle", "busy", "terminating", "restarting", "autorestarting", "dead"}
	ErrEmptyGatewayAddr = errors.New("cluster gateway IP address cannot be the empty string")
)

type KernelRefreshCallback func([]*gateway.DistributedJupyterKernel)

// Used to pass errors back to another window.
type ErrorHandler interface {
	HandleError(error, string)
}

type WorkloadDriver interface {
	// Return true if we're connected to the Cluster Gateway.
	ConnectedToGateway() bool

	KernelSpecProvider() KernelSpecProvider // Return the entity responsible for providing the up-to-date list of Jupyter kernel specs.
	KernelProvider() KernelProvider         // Return the entity responsible for providing the up-to-date list of Jupyter kernels.
	NodeProvider() NodeProvider             // Return the entity responsible for providing the up-to-date list of Kubernetes nodes.

	// Tell the Cluster Gateway to migrate a particular replica.
	MigrateKernelReplica(*gateway.MigrationRequest) error
	DialGatewayGRPC(string) error // Attempt to connect to the Cluster Gateway's gRPC server using the provided address. Returns an error if connection failed, or nil on success. This should NOT be called from the UI goroutine.
}

type WorkloadDriverOptions struct {
	HttpPort int `name:"http_port" description:"Port that the server will listen on." json:"http_port"`
}

type ResourceProvider[resource any] interface {
	Count() int32          // Number of currently-active resources.
	Resources() []resource // List of currently-active resources.
	RefreshResources()     // Manually/explicitly refresh the set of active resources from the Cluster Gateway.
	Start(string) error    // Start querying for resources periodically.

	RefreshOccurred()                                   // Called automatically when a refresh occurred; informs the subscribers.
	QueryResources()                                    // Call in its own goroutine; polls for resources.
	SubscribeToRefreshes(string, func([]resource) bool) // Subscribe to Kernel refreshes.
	UnsubscribeFromRefreshes(string)                    // Unsubscribe from Kernel refreshes.
	DialGatewayGRPC(string) error                       // Attempt to connect to the Cluster Gateway's gRPC server using the provided address. Returns an error if connection failed, or nil on success. This should NOT be called from the UI goroutine.
}

type KernelProvider interface {
	ResourceProvider[*gateway.DistributedJupyterKernel]
}

type KernelSpecProvider interface {
	ResourceProvider[*KernelSpec]
}

type NodeProvider interface {
	ResourceProvider[*KubernetesNode]
}

type KubernetesNode struct {
	NodeId          string           `json:"Nodes"`
	Pods            []*KubernetesPod `json:"Pods"`
	Age             time.Duration    `json:"Age"`
	IP              string           `json:"IP"`
	CapacityCPU     float64          `json:"CapacityCPU"`
	CapacityMemory  float64          `json:"CapacityMemory"`
	CapacityGPUs    float64          `json:"CapacityGPUs"`
	CapacityVGPUs   float64          `json:"CapacityVGPUs"`
	AllocatedCPU    float64          `json:"AllocatedCPU"`
	AllocatedMemory float64          `json:"AllocatedMemory"`
	AllocatedGPUs   float64          `json:"AllocatedGPUs"`
	AllocatedVGPUs  float64          `json:"AllocatedVGPUs"`

	Valid bool `json:"Valid"` // Used to determine if the struct was sent/received correctly over the network.
}

func (kn *KubernetesNode) String() string {
	out, err := json.Marshal(kn)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type KubernetesPod struct {
	PodName  string        `json:"PodName"`
	PodPhase string        `json:"PodPhase"`
	PodAge   time.Duration `json:"PodAge"`
	PodIP    string        `json:"PodIP"`

	Valid bool `json:"Valid"` // Used to determine if the struct was sent/received correctly over the network.
}

func (kp *KubernetesPod) String() string {
	out, err := json.Marshal(kp)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type ErrorMessage struct {
	ErrorMessage string `json:"ErrorMessage"`
	Valid        bool   `json:"Valid"` // Used to determine if the struct was sent/received correctly over the network.
}

func (m *ErrorMessage) String() string {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type BackendHttpHandler interface {
	http.Handler

	// Write an error back to the client.
	WriteError(*websocket.Conn, string)

	// Handle a message/request from the front-end.
	HandleRequest(*websocket.Conn, *http.Request, map[string]interface{})
}

type KernelSpec struct {
	Name              string             `json:"name"`
	DisplayName       string             `json:"display_name"`
	Language          string             `json:"language"`
	InterruptMode     string             `json:"interrupt_mode"`
	KernelProvisioner *KernelProvisioner `json:"kernel_provisioner"`
	ArgV              []string           `json:"argv"`
}

func (ks *KernelSpec) String() string {
	out, err := json.Marshal(ks)
	if err != nil {
		panic(err)
	}

	return string(out)
}

type KernelProvisioner struct {
	Name    string `json:"name"`
	Gateway string `json:"display_name"`
}

func (kp *KernelProvisioner) String() string {
	out, err := json.Marshal(kp)
	if err != nil {
		panic(err)
	}

	return string(out)
}
