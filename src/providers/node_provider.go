package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type BaseNodeProvider struct {
	domain.NodeProvider

	// rpcClient            gateway.ClusterGatewayClient // gRPC client to the Cluster Gateway.
	nodes                *cmap.ConcurrentMap[string, *domain.KubernetesNode]
	lastNodeRefresh      time.Time           // The last time we refreshed the nodes.
	lastNodeRefreshMutex sync.Mutex          // Sychronizes access to the lastNodeRefresh variable.
	refreshNodeMutex     sync.Mutex          // Sychrnoize node refreshes.
	nodeQueryTicker      *time.Ticker        // Sends ticks to retrieve updates on the nodes.
	quitNodeQueryChannel chan struct{}       // Used to tell the node querier (a goroutine) to stop querying.
	nodeQueryInterval    time.Duration       // How frequently to query the Gateway for Node updates. TODO(Ben): Eventually, the Gateway will publish this info to us proactively.
	errorHandler         domain.ErrorHandler // Pass errors here to be displayed to the user.
	connectedToGateway   bool                // Indicates whether or not we're connected to the Cluster Gateway
	gatewayAddress       string              // Address of the Cluster Gateway.
	spoofCluster         bool

	subscribers *cmap.ConcurrentMap[string, func([]*domain.KubernetesNode)]
}

func NewNodeProvider(nodeQueryInterval time.Duration, errorHandler domain.ErrorHandler, spoofCluster bool) domain.NodeProvider {
	nodes := cmap.New[*domain.KubernetesNode]()
	subscribers := cmap.New[func([]*domain.KubernetesNode)]()

	app.Logf("Will be querying and refreshing nodes every %v", nodeQueryInterval)

	baseProvider := &BaseNodeProvider{
		nodes:                &nodes,
		errorHandler:         errorHandler,
		quitNodeQueryChannel: make(chan struct{}),
		nodeQueryTicker:      time.NewTicker(nodeQueryInterval),
		nodeQueryInterval:    nodeQueryInterval,
		subscribers:          &subscribers,
		spoofCluster:         spoofCluster,
	}

	// if spoofCluster {
	// 	provider := newSpoofedNodeProvider(nodeQueryInterval, errorHandler, baseProvider)
	// 	return provider
	// } else {
	// 	baseProvider.NodeProvider = baseProvider
	// 	return baseProvider
	// }

	baseProvider.NodeProvider = baseProvider
	return baseProvider
}

func (p *BaseNodeProvider) SubscribeToRefreshes(key string, handler func([]*domain.KubernetesNode)) {
	p.subscribers.Set(key, handler)
}

func (p *BaseNodeProvider) UnsubscribeFromRefreshes(key string) {
	p.subscribers.Remove(key)
}

func (p *BaseNodeProvider) Start(addr string) error {
	if err := p.NodeProvider.DialGatewayGRPC(addr); err != nil {
		return err
	}
	p.gatewayAddress = addr

	go p.queryNodes()

	return nil
}

func (p *BaseNodeProvider) DialGatewayGRPC(gatewayAddress string) error {
	return nil
}

// Periodically query the Gateway for an update of the current active nodes.
// This should be called from its own goroutine.
//
// TODO(Ben): Eventually, the Gateway will publish this info to us proactively.
func (p *BaseNodeProvider) queryNodes() {
	for {
		select {
		case <-p.nodeQueryTicker.C:
			p.lastNodeRefreshMutex.Lock()
			// If we've manually refreshed the nodes since the last query interval, then we'll just wait to do our refresh.
			// This is basically checking if we've refreshed manually anytime recently.
			if time.Since(p.lastNodeRefresh) < p.nodeQueryInterval {
				p.lastNodeRefreshMutex.Unlock()
				continue
			}
			p.lastNodeRefreshMutex.Unlock()

			if !p.connectedToGateway {
				app.Log("[WARNING] Disconnected from Gateway; cannot query for node updates.")
				return
			}

			p.NodeProvider.RefreshResources()
		case <-p.quitNodeQueryChannel:
			app.Log("Ceasing node queries to Gateway.")
			return
		}
	}
}

func (p *BaseNodeProvider) Count() int32 {
	return int32(p.nodes.Count())
}

func (p *BaseNodeProvider) Resources() []*domain.KubernetesNode {
	nodes := make([]*domain.KubernetesNode, 0, p.Count())

	for kvPair := range p.nodes.IterBuffered() {
		nodes = append(nodes, kvPair.Val)
	}

	return nodes
}

// Map from NodeID to *gateway.KubernetesNode of currently-active nodes.
func (p *BaseNodeProvider) Nodes() *cmap.ConcurrentMap[string, *domain.KubernetesNode] {
	return p.nodes
}

// Actually refresh the nodes.
// This must not be called from the UI goroutine.
// This MUST be called with the p.refreshNodeMutex help.
// It is the caller's responsibility to release the lock afterwards.
func (p *BaseNodeProvider) RefreshResources() {
	locked := p.refreshNodeMutex.TryLock()
	if !locked {
		// If we did not acquire the lock, then there's already an active refresh occurring. We'll just return.
		app.Log("There is already an active refresh operation being performep. Please wait for it to complete.")
		return
	}
	defer p.refreshNodeMutex.Unlock()

	app.Log("Node Querier is refreshing nodes now.")

	ctxConnect, cancelConnect := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelConnect()
	c, _, err := websocket.Dial(ctxConnect, "ws://localhost:9995/api/k8s-nodes", nil)
	if err != nil {
		app.Logf("Failed to connect to backend while trying to refresh k8s nodes: %v", err)
		p.errorHandler.HandleError(err, "Failed to fetch list of active nodes from the Cluster Gateway. Could not connect to the backend.")
		return
	}
	defer c.CloseNow()

	msg := map[string]interface{}{
		"op":          "request-nodes",
		"spoof-nodes": p.spoofCluster,
	}

	ctxWrite, cancelWrite := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelWrite()
	err = wsjson.Write(ctxWrite, c, msg)
	if err != nil {
		p.errorHandler.HandleError(err, "Failed to fetch list of active nodes from the Cluster Gateway.")
		return
	}

	ctxRead, cancelRead := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelRead()
	var nodes map[string]*domain.KubernetesNode
	err = wsjson.Read(ctxRead, c, &nodes)
	c.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		app.Logf("Error encountered while reading nodes from backend: %v", err)
		return
	}

	app.Logf("Received from the backend: %v", nodes)

	// Clear the current nodes.
	p.nodes.Clear()
	for nodeName, node := range nodes {
		p.nodes.Set(nodeName, node)
	}

	for kv := range p.nodes.IterBuffered() {
		app.Log(fmt.Sprintf("Discovered active node %s: %s", kv.Key, kv.Val.String()))
	}

	p.refreshOccurred()
}

func (p *BaseNodeProvider) refreshOccurred() {
	p.lastNodeRefresh = time.Now()
	nodes := p.Resources()

	for kv := range p.subscribers.IterBuffered() {
		handler := kv.Val
		handler(nodes)
	}
}
