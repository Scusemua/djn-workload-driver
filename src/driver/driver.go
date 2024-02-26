package driver

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	cmap "github.com/orcaman/concurrent-map/v2"
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
	// kernels                *cmap.ConcurrentMap[string, *gateway.DistributedJupyterKernel] // Currently-active Jupyter kernels (that we know about). Map from Kernel ID to Kernel.
	nodes                  *cmap.ConcurrentMap[string, *gateway.KubernetesNode] // Currently-active Kubernetes nodes. Map from node ID to node.
	errorHandler           domain.ErrorHandler                                  // Pass errors here to be displayed to the user.
	spoofGatewayConnection bool                                                 // Used for development when not actually using a real cluster.
	nodeQueryInterval      time.Duration                                        // How frequently to query the Gateway for node updates.
	rpcCallTimeout         time.Duration                                        // Timeout for individual RPC calls.
	lastNodesRefresh       time.Time                                            // The last time we refreshed the nodes.
	lastNodesRefreshMutex  sync.Mutex                                           // Sychronizes access to the lastNodesRefresh variable.
	refreshNodeMutex       sync.Mutex                                           // Synchronizes node refreshes.
	rpcClient              gateway.ClusterGatewayClient                         // gRPC client to the Cluster Gateway.

	kernelProvider domain.KernelProvider

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
	nodeMap := cmap.New[*gateway.KubernetesNode]()
	driver := &workloadDriverImpl{
		// kernels:                &kernelMap,
		nodes:                  &nodeMap,
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

	return driver
}

func (d *workloadDriverImpl) queryNodes() {
	for {
		select {
		case <-d.nodeQueryTicker.C:
			d.lastNodesRefreshMutex.Lock()

			if time.Since(d.lastNodesRefresh) < d.nodeQueryInterval {
				d.lastNodesRefreshMutex.Unlock()
				continue
			}
			d.lastNodesRefreshMutex.Unlock()

			if !d.connectedToGateway {
				app.Log("[WARNING] Disconnected from Gateway; cannot query for node updates.")
				return
			}

			app.Log("Node Querier is refreshing kernels now.")
			d.refreshNodeMutex.Lock()
			d.refreshNodes()
			d.refreshNodeMutex.Unlock()
		case <-d.quitNodeQueryTicker:
			app.Log("Ceasing node queries to Gateway.")
			return
		}
	}
}

// Should not be called from UI goroutine.
// Must be called with the refreshKernelMutex held.
func (d *workloadDriverImpl) refreshNodes() {
	app.Log("Refreshing nodes.")

	if d.spoofGatewayConnection {
		// TODO(Ben): Spoof nodes?

		// Simulate some delay.
		delay_ms := rand.Int31n(1500)

		app.Logf("Sleeping for %d milliseconds.", delay_ms)
		time.Sleep(time.Millisecond * time.Duration(delay_ms))
	} else {
		app.Log("Fetching nodes now.")
		resp, err := d.rpcClient.GetKubernetesNodes(context.TODO(), &gateway.Void{})
		if err != nil {
			d.errorHandler.HandleError(err, "Failed to fetch list of active kernels from the Cluster Gateway.")
		}

		// Clear the current kernels.
		d.nodes.Clear()
		for _, node := range resp.Nodes {
			d.nodes.Set(node.NodeId, node)
		}
	}

	d.lastNodesRefresh = time.Now()
}

func (d *workloadDriverImpl) Start() {
	// If we're not spoofing the Gateway connection, then periodically query the Gateway for an update of the current active kernels.
	app.Log("Starting Gateway Querier now.")
	d.kernelProvider.Start()
	go d.queryNodes()
}

func (d *workloadDriverImpl) ConnectedToGateway() bool {
	return d.connectedToGateway
}

// This should NOT be called from the UI goroutine.
func (d *workloadDriverImpl) DialGatewayGRPC(gatewayAddress string) error {
	d.kernelProvider.DialGatewayGRPC(gatewayAddress)

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

	return nil
}

// Return a list of currently-active kernels.
func (d *workloadDriverImpl) KernelsSlice() []*gateway.DistributedJupyterKernel {
	return d.kernelProvider.KernelsSlice()
}

// Return a list of currently-active kernels.
func (d *workloadDriverImpl) NumKernels() int32 {
	return d.kernelProvider.NumKernels()
}

// Manually/explicitly refresh the set of active kernels from the Cluster Gateway.
func (d *workloadDriverImpl) RefreshKernels() {
	d.kernelProvider.RefreshKernels()
}

func (d *workloadDriverImpl) GetKubernetesNodes() ([]*gateway.KubernetesNode, error) {
	return nil, nil
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
