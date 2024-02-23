package domain

import (
	"errors"
)

var (
	ErrEmptyGatewayAddr = errors.New("cluster gateway IP address cannot be the empty string")
)

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
}

type WorkloadDriverOptions struct {
	HttpPort int `name:"http_port" description:"Port that the server will listen on." json:"http_port"`
}

type Kernel interface {
	GetKernelId() string
	GetNumReplicas() int32
	GetStatus() string
	GetAggregateBusyStatus() string
}

type KernelProvider interface {
	NumKernels() int32      // Number of currently-active kernels.
	Kernels() []Kernel      // List of currently-active kernels.
	GatewayAddress() string // Return the address of the Cluster Gateway from which the list of kernels was retrieved.
}
