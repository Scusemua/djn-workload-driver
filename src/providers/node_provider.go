package providers

import (
	"context"
	"time"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type BaseNodeProvider struct {
	*BaseProvider[*domain.KubernetesNode]
}

func NewNodeProvider(nodeQueryInterval time.Duration, errorHandler domain.ErrorHandler, spoofCluster bool) domain.NodeProvider {
	app.Logf("Will be querying and refreshing nodes every %v", nodeQueryInterval)

	// If we're spoofing the cluster, then don't connect to the Gateway.
	var doConnectToGateway bool = !spoofCluster

	// Create the base provider that provides implementations to methods common to all types of resource providers.
	baseProvider := newBaseProvider[*domain.KubernetesNode](nodeQueryInterval, errorHandler, doConnectToGateway)

	// Create the NodeProvider.
	nodeProvider := &BaseNodeProvider{
		BaseProvider: baseProvider,
	}

	nodeProvider.ResourceProvider = nodeProvider
	return nodeProvider
}

// Actually refresh the nodes.
// This must not be called from the UI goroutine.
// This MUST be called with the p.refreshNodeMutex help.
// It is the caller's responsibility to release the lock afterwards.
func (p *BaseNodeProvider) RefreshResources() {
	locked := p.refreshMutex.TryLock()
	if !locked {
		// If we did not acquire the lock, then there's already an active refresh occurring. We'll just return.
		app.Log("There is already an active refresh operation being performep. Please wait for it to complete.")
		return
	}
	defer p.refreshMutex.Unlock()

	app.Log("Node Querier is refreshing nodes now.")

	ctxConnect, cancelConnect := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelConnect()
	c, _, err := websocket.Dial(ctxConnect, "ws://localhost:8000"+domain.KUBERNETES_NODES_ENDPOINT, nil)
	if err != nil {
		app.Logf("Failed to connect to backend while trying to refresh k8s nodes: %v", err)
		p.errorHandler.HandleError(err, "Failed to fetch list of active nodes from the Cluster Gateway. Could not connect to the backend.")
		return
	}
	defer c.CloseNow()

	msg := map[string]interface{}{
		"op": "request-nodes",
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

	// app.Logf("Received from the backend: %v", nodes)

	// Clear the current nodes.
	p.resources.Clear()
	for nodeName, node := range nodes {
		p.resources.Set(nodeName, node)
	}

	p.RefreshOccurred()
}
