package providers

import (
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

type BaseProvider[Resource any] struct {
	domain.ResourceProvider[Resource]

	rpcClient           gateway.ClusterGatewayClient          // gRPC client to the Cluster Gateway.
	resources           *cmap.ConcurrentMap[string, Resource] // Latest resources.
	lastRefresh         time.Time                             // The last time we refreshed the resources.
	lastRefreshMutex    sync.Mutex                            // Sychronizes access to the lastResourceRefresh variable.
	refreshMutex        sync.Mutex                            // Sychrnoize resource refreshes.
	resourceQueryTicker *time.Ticker                          // Sends ticks to retrieve updates on the resources.
	quitQueryChannel    chan struct{}                         // Used to tell the resource querier (a goroutine) to stop querying.
	queryInterval       time.Duration                         // How frequently to query the Gateway for resource updates. TODO(Ben): Eventually, the Gateway will publish this info to us proactively.
	errorHandler        domain.ErrorHandler                   // Pass errors here to be displayed to the user.
	connectedToGateway  bool                                  // Indicates whether or not we're connected to the Cluster Gateway
	gatewayAddress      string                                // Address of the Cluster Gateway.
	doConnectToGateway  bool                                  // True if this provider should actually attempt to connect to the gateway. Some providers don't need to.

	subscribers *cmap.ConcurrentMap[string, func([]Resource)]
}

func newBaseProvider[Resource any](queryInterval time.Duration, errorHandler domain.ErrorHandler, doConnectToGateway bool) *BaseProvider[Resource] {
	resources := cmap.New[Resource]()
	subscribers := cmap.New[func([]Resource)]()

	provider := &BaseProvider[Resource]{
		doConnectToGateway:  doConnectToGateway,
		resources:           &resources,
		subscribers:         &subscribers,
		queryInterval:       queryInterval,
		errorHandler:        errorHandler,
		resourceQueryTicker: time.NewTicker(queryInterval),
		quitQueryChannel:    make(chan struct{}),
	}

	provider.ResourceProvider = provider

	return provider
}

// Number of currently-active resources.
func (p *BaseProvider[Resource]) Count() int32 {
	return int32(p.resources.Count())
}

// List of currently-active resources.
func (p *BaseProvider[Resource]) Resources() []Resource {
	resources := make([]Resource, 0, p.Count())

	for kvPair := range p.resources.IterBuffered() {
		resources = append(resources, kvPair.Val)
	}

	return resources
}

func (p *BaseProvider[Resource]) RefreshOccurred() {
	p.lastRefresh = time.Now()
	resources := p.Resources()

	for kv := range p.subscribers.IterBuffered() {
		handler := kv.Val
		handler(resources)
	}
}

// Periodically query the Gateway for an update of the current active kernels.
// This should be called from its own goroutine.
func (p *BaseProvider[Resource]) QueryResources() {
	for {
		select {
		case <-p.resourceQueryTicker.C:
			p.lastRefreshMutex.Lock()
			// If we've manually refreshed the kernels since the last query interval, then we'll just wait to do our refresh.
			// This is basically checking if we've refreshed manually anytime recently.
			if time.Since(p.lastRefresh) < p.queryInterval {
				p.lastRefreshMutex.Unlock()
				continue
			}
			p.lastRefreshMutex.Unlock()

			if !p.connectedToGateway {
				app.Log("[WARNING] Disconnected from Gateway; cannot query for kernel updates.")
				return
			}

			p.ResourceProvider.RefreshResources()
		case <-p.quitQueryChannel:
			app.Log("Ceasing kernel queries to Gateway.")
			return
		}
	}
}

// Manually/explicitly refresh the set of active resources from the Cluster Gateway.
func (p *BaseProvider[Resource]) RefreshResources() {
	p.ResourceProvider.RefreshResources()
}

// Start querying for resources periodically.
func (p *BaseProvider[Resource]) Start(addr string) error {
	err := p.ResourceProvider.DialGatewayGRPC(addr)
	if err != nil {
		return err
	}

	p.gatewayAddress = addr

	go p.ResourceProvider.QueryResources()

	return nil
}

// Subscribe to Kernel refreshes.
func (p *BaseProvider[Resource]) SubscribeToRefreshes(id string, handler func([]Resource)) {
	p.subscribers.Set(id, handler)
}

// Unsubscribe from Kernel refreshes.
func (p *BaseProvider[Resource]) UnsubscribeFromRefreshes(id string) {
	p.subscribers.Remove(id)
}

// Attempt to connect to the Cluster Gateway's gRPC server using the provided address. Returns an error if connection failed, or nil on success. This should NOT be called from the UI goroutine.
func (p *BaseProvider[Resource]) DialGatewayGRPC(gatewayAddress string) error {
	// Return immediately if we're not supposed to connect.
	if !p.doConnectToGateway {
		app.Logf("Spoofing RPC connection to Cluster Gateway...")
		time.Sleep(time.Second * 1)
		p.connectedToGateway = true
		p.gatewayAddress = gatewayAddress
		return nil
	}

	if gatewayAddress == "" {
		return domain.ErrEmptyGatewayAddr
	}

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

	return nil
}
