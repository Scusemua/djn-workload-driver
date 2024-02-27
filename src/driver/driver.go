package driver

import (
	"context"
	"errors"
	"time"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
	"github.com/scusemua/djn-workload-driver/m/v2/src/config"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
	"github.com/scusemua/djn-workload-driver/m/v2/src/providers"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrRequestIgnoredCxnSpoofed = errors.New("migration operation cannot be performed as the connection to the Cluster Gateway is spoofed")
	ErrRpcDisconnected          = errors.New("cannot perform the requested RPC as we are not connected to the Cluster Gateway")
)

type workloadDriverImpl struct {
	connectedToGateway bool   // Flag indicating whether or not we're currently connected to the Cluster Gateway.
	gatewayAddress     string // IP address of the Gateway.

	errorHandler           domain.ErrorHandler          // Pass errors here to be displayed to the user.
	spoofGatewayConnection bool                         // Used for development when not actually using a real cluster.
	nodeQueryInterval      time.Duration                // How frequently to query the Gateway for node updates.
	rpcCallTimeout         time.Duration                // Timeout for individual RPC calls.
	rpcClient              gateway.ClusterGatewayClient // gRPC client to the Cluster Gateway.

	kernelProvider domain.KernelProvider
	nodeProvider   domain.NodeProvider

	nodeQueryTicker     *time.Ticker  // Sends ticks to retrieve updates on the nodes.
	quitNodeQueryTicker chan struct{} // Used to tell the node querier (a goroutine) to stop querying.
}

func NewWorkloadDriver(errorHandler domain.ErrorHandler, opts *config.Configuration) *workloadDriverImpl {
	kernelQueryInterval, err := time.ParseDuration(opts.KernelQueryInterval)
	if err != nil {
		panic(err)
	}

	nodeQueryInterval, err := time.ParseDuration(opts.NodeQueryInterval)
	if err != nil {
		panic(err)
	}

	// kernelMap := cmap.New[*gateway.DistributedJupyterKernel]()
	// nodeMap := cmap.New[*domain.KubernetesNode]()
	driver := &workloadDriverImpl{
		// kernels:                &kernelMap,
		// nodes:                  &nodeMap,
		errorHandler:           errorHandler,
		spoofGatewayConnection: opts.SpoofCluster,
		nodeQueryInterval:      nodeQueryInterval,
		nodeQueryTicker:        time.NewTicker(nodeQueryInterval),
		quitNodeQueryTicker:    make(chan struct{}),
	}

	if driver.spoofGatewayConnection {
		driver.kernelProvider = providers.NewSpoofedKernelProvider(kernelQueryInterval, errorHandler)
	} else {
		driver.kernelProvider = providers.NewKernelProvider(kernelQueryInterval, errorHandler)
	}

	driver.nodeProvider = providers.NewNodeProvider(nodeQueryInterval, errorHandler, opts.SpoofCluster)

	return driver
}

func (d *workloadDriverImpl) Start(addr string) error {
	// Do nothing.
	return nil
}

func (d *workloadDriverImpl) ConnectedToGateway() bool {
	return d.connectedToGateway
}

// This should NOT be called from the UI goroutine.
func (d *workloadDriverImpl) DialGatewayGRPC(gatewayAddress string) error {
	if d.spoofGatewayConnection {
		time.Sleep(time.Second * 1)
		d.connectedToGateway = true
		d.gatewayAddress = gatewayAddress
	} else if gatewayAddress == "" {
		return domain.ErrEmptyGatewayAddr
	} else {
		// p.logger.Info(fmt.Sprintf("Attempting to dial Gateway gRPC server now. Address: %s\n", p.gatewayAddress))
		app.Logf("Attempting to dial Gateway gRPC server now. Address: %s\n", gatewayAddress)

		ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
		defer cancel()
		conn, err := grpc.DialContext(ctx, gatewayAddress, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err != nil {
			// p.logger.Error(fmt.Sprintf("Failed to dial Gateway gRPC server. Address: %s. Error: %v.\n", p.gatewayAddress, zap.Error(err)))
			app.Logf("Failed to dial Gateway gRPC server. Address: %s. Error: %v.\n", gatewayAddress, zap.Error(err))
			return err
		}

		// p.logger.Info(fmt.Sprintf("Successfully dialed Cluster Gateway at address %s.\n", p.gatewayAddress))
		app.Logf("Successfully dialed Cluster Gateway at address %s.\n", gatewayAddress)
		d.rpcClient = gateway.NewClusterGatewayClient(conn)
		d.connectedToGateway = true
		d.gatewayAddress = gatewayAddress
	}

	app.Log("Starting Gateway Querier now.")
	d.kernelProvider.Start(gatewayAddress)
	d.nodeProvider.Start(gatewayAddress)

	return nil
}

// Return a list of currently-active kernels.
func (d *workloadDriverImpl) Resources() []*gateway.DistributedJupyterKernel {
	return d.kernelProvider.Resources()
}

// Return a list of currently-active kernels.
func (d *workloadDriverImpl) Count() int32 {
	return d.kernelProvider.Count()
}

// Manually/explicitly refresh the set of active kernels from the Cluster Gateway.
func (d *workloadDriverImpl) RefreshResources() {
	d.kernelProvider.RefreshResources()
}

func (d *workloadDriverImpl) KernelProvider() domain.KernelProvider {
	return d.kernelProvider
}

func (d *workloadDriverImpl) NodeProvider() domain.NodeProvider {
	return d.nodeProvider
}

func (d *workloadDriverImpl) MigrateKernelReplica(arg *gateway.MigrationRequest) error {
	if d.spoofGatewayConnection {
		app.Log("[WARNING] We're spoofing the connection to the Gateway. Ignoring migration request.")
		return ErrRequestIgnoredCxnSpoofed
	}

	if !d.connectedToGateway {
		app.Log("[ERROR] Cannot perform migration operation as we're not connected to the Cluster Gateway.")
		return ErrRpcDisconnected
	}

	if arg == nil {
		panic("Received nil argument for call to MigrateKernelReplica")
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.rpcCallTimeout)
	defer cancel()
	resp, err := d.rpcClient.MigrateKernelReplica(ctx, arg)

	if err != nil {
		app.Logf("[ERROR] Recevied error in response to MigrateKernelReplica: %v", err)
		return err
	}

	app.Logf("Response for MigrateKernelReplica requqest: %v", resp)

	return nil
}

// Subscribe to Kernel refreshes.
func (d *workloadDriverImpl) SubscribeToRefreshes(key string, handler func([]*gateway.DistributedJupyterKernel)) {
	d.kernelProvider.SubscribeToRefreshes(key, handler)
}

// Unsubscribe from Kernel refreshes.
func (d *workloadDriverImpl) UnsubscribeFromRefreshes(key string) {
	d.kernelProvider.UnsubscribeFromRefreshes(key)
}

func (d *workloadDriverImpl) GatewayAddress() string {
	return d.gatewayAddress
}
