package domain

import (
	"errors"

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
	KernelProvider

	// Start the WorkloadDriver
	Start()

	// Return true if we're connected to the Cluster Gateway.
	ConnectedToGateway() bool

	// Tell the Cluster Gateway to migrate a particular replica.
	MigrateKernelReplica(*gateway.MigrationRequest) error

	// Return a list of the Kubernetes nodes available within the Kubernetes cluster.
	GetKubernetesNodes() ([]*gateway.KubernetesNode, error)

	GatewayAddress() string // Return the address of the Cluster Gateway from which the list of kernels was retrieved.
}

type WorkloadDriverOptions struct {
	HttpPort int `name:"http_port" description:"Port that the server will listen on." json:"http_port"`
}

type KernelProvider interface {
	NumKernels() int32                                                      // Number of currently-active kernels.
	Kernels() []*gateway.DistributedJupyterKernel                           // List of currently-active kernels.
	RefreshKernels()                                                        // Manually/explicitly refresh the set of active kernels from the Cluster Gateway.
	Start()                                                                 // Start querying for kernels periodically.
	DialGatewayGRPC(string) error                                           // Attempt to connect to the Cluster Gateway's gRPC server using the provided address. Returns an error if connection failed, or nil on success. This should NOT be called from the UI goroutine.
	SubscribeToRefreshes(string, func([]*gateway.DistributedJupyterKernel)) // Subscribe to Kernel refreshes.
	UnsubscribeFromRefreshes(string)                                        // Unsubscribe from Kernel refreshes.
}
