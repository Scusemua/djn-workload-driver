package domain

import (
	"encoding/json"
	"errors"
	"time"

	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
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

	KernelProvider() KernelProvider
	NodeProvider() NodeProvider

	// Tell the Cluster Gateway to migrate a particular replica.
	MigrateKernelReplica(*gateway.MigrationRequest) error

	GatewayAddress() string // Return the address of the Cluster Gateway from which the list of kernels was retrieved.

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

	SubscribeToRefreshes(string, func([]resource)) // Subscribe to Kernel refreshes.
	UnsubscribeFromRefreshes(string)               // Unsubscribe from Kernel refreshes.
	DialGatewayGRPC(string) error                  // Attempt to connect to the Cluster Gateway's gRPC server using the provided address. Returns an error if connection failed, or nil on success. This should NOT be called from the UI goroutine.
}

type KernelProvider interface {
	ResourceProvider[*gateway.DistributedJupyterKernel]
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
}

type KubernetesPod struct {
	PodName  string        `json:"PodName"`
	PodPhase string        `json:"PodPhase"`
	PodAge   time.Duration `json:"PodAge"`
	PodIP    string        `json:"PodIP"`
}

func (kn *KubernetesNode) String() string {
	out, err := json.Marshal(kn)
	if err != nil {
		panic(err)
	}

	return string(out)
}
