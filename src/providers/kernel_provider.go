package providers

import (
	"context"
	"time"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
)

type BaseKernelProvider struct {
	*BaseProvider[*gateway.DistributedJupyterKernel]
}

func NewKernelProvider(kernelQueryInterval time.Duration, errorHandler domain.ErrorHandler) domain.KernelProvider {
	// Create the base provider that provides implementations to methods common to all types of resource providers.
	baseProvider := newBaseProvider[*gateway.DistributedJupyterKernel](kernelQueryInterval, errorHandler, true)

	// Create the KernelProvider.
	provider := &BaseKernelProvider{
		BaseProvider: baseProvider,
	}

	provider.ResourceProvider = provider

	app.Logf("Will be querying and refreshing kernels every %v", kernelQueryInterval)

	return provider
}

// Actually refresh the kernels.
// This must not be called from the UI goroutine.
// This MUST be called with the p.refreshKernelMutex help.
// It is the caller's responsibility to release the lock afterwards.
func (p *BaseKernelProvider) RefreshResources() {
	locked := p.refreshMutex.TryLock()
	if !locked {
		// If we did not acquire the lock, then there's already an active refresh occurring. We'll just return.
		app.Log("There is already an active refresh operation being performep. Please wait for it to complete.")
		return
	}
	defer p.refreshMutex.Unlock()

	app.Log("Kernel Querier is refreshing kernels now.")
	resp, err := p.rpcClient.ListKernels(context.TODO(), &gateway.Void{})
	if err != nil || resp == nil {
		app.Logf("[ERROR] Failed to fetch list of active kernels from the Cluster Gateway: %v.", err)
		p.errorHandler.HandleError(err, "Failed to fetch list of active kernels from the Cluster Gateway.")
		return
	}

	// Clear the current kernels.
	p.resources.Clear()
	for _, kernel := range resp.Kernels {
		p.resources.Set(kernel.KernelId, kernel)
		app.Log("Discovered active kernel! ID=%s, NumReplicas=%d, Status1=%s, Status2=%s", kernel.KernelId, kernel.NumReplicas, kernel.Status, kernel.AggregateBusyStatus)
	}

	p.RefreshOccurred()
}
