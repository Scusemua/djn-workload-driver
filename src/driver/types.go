package driver

import (
	"errors"

	"github.com/scusemua/djn-workload-driver/m/v2/src/gateway"
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

	// Attempt to connect to the Cluster Gateway's gRPC server using the provided address.
	// Returns an error if connection failed, or nil on success.
	DialGatewayGRPC(string) error

	// Tell the Cluster Gateway to migrate a particular replica.
	MigrateKernelReplica(*gateway.MigrationRequest) error
}

type WorkloadDriverOptions struct {
	HttpPort int `name:"http_port" description:"Port that the server will listen on." json:"http_port"`
}

type KernelProvider interface {
	NumKernels() int32                            // Number of currently-active kernels.
	Kernels() []*gateway.DistributedJupyterKernel // List of currently-active kernels.
	GatewayAddress() string                       // Return the address of the Cluster Gateway from which the list of kernels was retrieved.
	ManuallyRefreshKernels()                      // Manually/explicitly refresh the set of active kernels from the Cluster Gateway.
}
