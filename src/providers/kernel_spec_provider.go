package providers

import (
	"context"
	"sort"
	"time"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type BaseKernelSpecProvider struct {
	*BaseProvider[*domain.KernelSpec]
}

func NewBaseKernelSpecProvider(kernelSpecQueryInterval time.Duration, errorHandler domain.ErrorHandler) *BaseKernelSpecProvider {
	provider := &BaseKernelSpecProvider{BaseProvider: newBaseProvider[*domain.KernelSpec](kernelSpecQueryInterval, errorHandler, false)}
	provider.ResourceProvider = provider
	return provider
}

// Manually/explicitly refresh the set of active resources from the Cluster Gateway.
func (p *BaseKernelSpecProvider) RefreshResources() {
	locked := p.refreshMutex.TryLock()
	if !locked {
		// If we did not acquire the lock, then there's already an active refresh occurring. We'll just return.
		app.Log("There is already an active refresh operation being performep. Please wait for it to complete.")
		return
	}
	defer p.refreshMutex.Unlock()

	app.Log("KernelSpec Querier is refreshing kernel specs now.")

	ctxConnect, cancelConnect := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelConnect()
	c, _, err := websocket.Dial(ctxConnect, "ws://localhost:9995"+domain.KERNEL_SPEC_ENDPOINT, nil)
	if err != nil {
		app.Logf("Failed to connect to backend while trying to refresh k8s kernel specs: %v", err)
		p.errorHandler.HandleError(err, "Failed to fetch list of active kernel specs from the Cluster Gateway. Could not connect to the backend.")
		return
	}
	defer c.CloseNow()

	msg := map[string]interface{}{
		"op": "request-kernel-specs",
	}

	ctxWrite, cancelWrite := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelWrite()
	err = wsjson.Write(ctxWrite, c, msg)
	if err != nil {
		p.errorHandler.HandleError(err, "Failed to fetch list of active kernel specs from the Cluster Gateway.")
		return
	}

	ctxRead, cancelRead := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelRead()
	var kernelSpecs []*domain.KernelSpec
	err = wsjson.Read(ctxRead, c, &kernelSpecs)
	c.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		app.Logf("Error encountered while reading kernel specs from backend: %v", err)
		return
	}

	app.Logf("Received kernel specs from the backend: %v", kernelSpecs)

	sort.Slice(kernelSpecs, func(i, j int) bool {
		return kernelSpecs[i].Name < kernelSpecs[j].Name
	})

	// Clear the current kernel specs.
	p.resources.Clear()
	for _, kernelSpec := range kernelSpecs {
		p.resources.Set(kernelSpec.Name, kernelSpec)
	}

	p.RefreshOccurred()
}
