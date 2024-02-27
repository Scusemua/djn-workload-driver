package providers

import (
	"context"
	"sync"
	"time"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	cmap "github.com/orcaman/concurrent-map/v2"
	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
	"github.com/scusemua/djn-workload-driver/m/v2/src/proxy"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type BaseKernelProvider struct {
	domain.KernelProvider

	rpcClient              gateway.ClusterGatewayClient // gRPC client to the Cluster Gateway.
	kernels                *cmap.ConcurrentMap[string, *gateway.DistributedJupyterKernel]
	lastKernelRefresh      time.Time           // The last time we refreshed the kernels.
	lastKernelRefreshMutex sync.Mutex          // Sychronizes access to the lastKernelRefresh variable.
	refreshKernelMutex     sync.Mutex          // Sychrnoize kernel refreshes.
	kernelQueryTicker      *time.Ticker        // Sends ticks to retrieve updates on the kernels.
	quitKernelQueryChannel chan struct{}       // Used to tell the kernel querier (a goroutine) to stop querying.
	kernelQueryInterval    time.Duration       // How frequently to query the Gateway for Kernel updates. TODO(Ben): Eventually, the Gateway will publish this info to us proactively.
	errorHandler           domain.ErrorHandler // Pass errors here to be displayed to the user.
	connectedToGateway     bool                // Indicates whether or not we're connected to the Cluster Gateway
	gatewayAddress         string              // Address of the Cluster Gateway.

	subscribers *cmap.ConcurrentMap[string, func([]*gateway.DistributedJupyterKernel)]
}

func NewKernelProvider(kernelQueryInterval time.Duration, errorHandler domain.ErrorHandler) *BaseKernelProvider {
	kernels := cmap.New[*gateway.DistributedJupyterKernel]()
	subscribers := cmap.New[func([]*gateway.DistributedJupyterKernel)]()

	provider := &BaseKernelProvider{
		kernels:                &kernels,
		errorHandler:           errorHandler,
		quitKernelQueryChannel: make(chan struct{}),
		kernelQueryTicker:      time.NewTicker(kernelQueryInterval),
		kernelQueryInterval:    kernelQueryInterval,
		subscribers:            &subscribers,
	}

	provider.KernelProvider = provider

	app.Logf("Will be querying and refreshing kernels every %v", kernelQueryInterval)

	return provider
}

func (p *BaseKernelProvider) SubscribeToRefreshes(key string, handler func([]*gateway.DistributedJupyterKernel)) {
	p.subscribers.Set(key, handler)
}

func (p *BaseKernelProvider) UnsubscribeFromRefreshes(key string) {
	p.subscribers.Remove(key)
}

func (p *BaseKernelProvider) DialGatewayGRPC(gatewayAddress string) error {
	if gatewayAddress == "" {
		return domain.ErrEmptyGatewayAddr
	} else {
		// p.logger.Info(fmt.Sprintf("Attempting to dial Gateway gRPC server now. Address: %s\n", p.gatewayAddress))
		app.Logf("Attempting to dial Gateway gRPC server now. Address: %s\n", gatewayAddress)

		webSocketProxyClient := proxy.NewWebSocketProxyClient(time.Minute)
		conn, err := grpc.Dial("ws://"+gatewayAddress, grpc.WithContextDialer(webSocketProxyClient.Dialer), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err != nil {
			// p.logger.Error(fmt.Sprintf("Failed to dial Gateway gRPC server. Address: %s. Error: %v.\n", p.gatewayAddress, zap.Error(err)))
			app.Logf("Failed to dial Gateway gRPC server. Address: %s. Error: %v.\n", gatewayAddress, zap.Error(err))
			return err
		}

		// p.logger.Info(fmt.Sprintf("Successfully dialed Cluster Gateway at address %s.\n", p.gatewayAddress))
		app.Logf("Successfully dialed Cluster Gateway at address %s.\n", gatewayAddress)
		p.rpcClient = gateway.NewClusterGatewayClient(conn)
		p.connectedToGateway = true
		p.gatewayAddress = gatewayAddress
	}

	return nil
}

func (p *BaseKernelProvider) Start(addr string) error {
	err := p.KernelProvider.DialGatewayGRPC(addr)
	if err != nil {
		return err
	}

	p.gatewayAddress = addr

	go p.queryKernels()

	return nil
}

// Periodically query the Gateway for an update of the current active kernels.
// This should be called from its own goroutine.
//
// TODO(Ben): Eventually, the Gateway will publish this info to us proactively.
func (p *BaseKernelProvider) queryKernels() {
	for {
		select {
		case <-p.kernelQueryTicker.C:
			p.lastKernelRefreshMutex.Lock()
			// If we've manually refreshed the kernels since the last query interval, then we'll just wait to do our refresh.
			// This is basically checking if we've refreshed manually anytime recently.
			if time.Since(p.lastKernelRefresh) < p.kernelQueryInterval {
				p.lastKernelRefreshMutex.Unlock()
				continue
			}
			p.lastKernelRefreshMutex.Unlock()

			if !p.connectedToGateway {
				app.Log("[WARNING] Disconnected from Gateway; cannot query for kernel updates.")
				return
			}

			p.KernelProvider.RefreshResources()
		case <-p.quitKernelQueryChannel:
			app.Log("Ceasing kernel queries to Gateway.")
			return
		}
	}
}

func (p *BaseKernelProvider) Count() int32 {
	return int32(p.kernels.Count())
}

func (p *BaseKernelProvider) Resources() []*gateway.DistributedJupyterKernel {
	kernels := make([]*gateway.DistributedJupyterKernel, 0, p.Count())

	for kvPair := range p.kernels.IterBuffered() {
		kernels = append(kernels, kvPair.Val)
	}

	return kernels
}

// Map from KernelID to *gateway.DistributedJupyterKernel of currently-active kernels.
func (p *BaseKernelProvider) Kernels() *cmap.ConcurrentMap[string, *gateway.DistributedJupyterKernel] {
	return p.kernels
}

// Actually refresh the kernels.
// This must not be called from the UI goroutine.
// This MUST be called with the p.refreshKernelMutex help.
// It is the caller's responsibility to release the lock afterwards.
func (p *BaseKernelProvider) RefreshResources() {
	locked := p.refreshKernelMutex.TryLock()
	if !locked {
		// If we did not acquire the lock, then there's already an active refresh occurring. We'll just return.
		app.Log("There is already an active refresh operation being performep. Please wait for it to complete.")
		return
	}
	defer p.refreshKernelMutex.Unlock()

	app.Log("Kernel Querier is refreshing kernels now.")
	resp, err := p.rpcClient.ListKernels(context.TODO(), &gateway.Void{})
	if err != nil || resp == nil {
		app.Logf("[ERROR] Failed to fetch list of active kernels from the Cluster Gateway: %v.", err)
		p.errorHandler.HandleError(err, "Failed to fetch list of active kernels from the Cluster Gateway.")
		return
	}

	// Clear the current kernels.
	p.kernels.Clear()
	for _, kernel := range resp.Kernels {
		p.kernels.Set(kernel.KernelId, kernel)
		app.Log("Discovered active kernel! ID=%s, NumReplicas=%d, Status1=%s, Status2=%s", kernel.KernelId, kernel.NumReplicas, kernel.Status, kernel.AggregateBusyStatus)
	}

	p.refreshOccurred()
}

func (p *BaseKernelProvider) refreshOccurred() {
	p.lastKernelRefresh = time.Now()
	kernels := p.Resources()

	for kv := range p.subscribers.IterBuffered() {
		handler := kv.Val
		handler(kernels)
	}
}
