package providers

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
)

type SpoofedKernelProvider struct {
	*BaseKernelProvider
}

func NewSpoofedKernelProvider(kernelQueryInterval time.Duration, errorHandler domain.ErrorHandler) domain.KernelProvider {
	// The BaseProvider will be created in the call to NewKernelProvider.
	baseKernelProvider := NewKernelProvider(kernelQueryInterval, errorHandler)

	provider := &SpoofedKernelProvider{
		BaseKernelProvider: baseKernelProvider.(*BaseKernelProvider),
	}

	provider.ResourceProvider = provider

	return provider
}

// Create an individual spoofed/fake kernel.
func (p *SpoofedKernelProvider) spoofKernel() *gateway.DistributedJupyterKernel {
	status := domain.KernelStatuses[rand.Intn(len(domain.KernelStatuses))]
	numReplicas := rand.Intn(5-2) + 2
	kernelId := uuid.New().String()
	// Spoof the kernel itself.
	kernel := &gateway.DistributedJupyterKernel{
		KernelId:            kernelId,
		NumReplicas:         int32(numReplicas),
		Status:              status,
		AggregateBusyStatus: status,
		Replicas:            make([]*gateway.JupyterKernelReplica, 0, numReplicas),
	}

	// Spoof the kernel's replicas.
	for j := 0; j < numReplicas; j++ {
		podId := fmt.Sprintf("kernel-%s-%s", kernelId, uuid.New().String()[0:5])
		replica := &gateway.JupyterKernelReplica{
			ReplicaId: int32(j),
			KernelId:  kernelId,
			PodId:     podId,
			NodeId:    fmt.Sprintf("Node-%d", rand.Intn(4-1)+1),
		}
		kernel.Replicas = append(kernel.Replicas, replica)
	}

	return kernel
}

// Called when spoofing kernels for the first time.
func (p *SpoofedKernelProvider) spoofInitialKernels() {
	numKernels := rand.Intn(8-2) + 2

	for i := 0; i < numKernels; i++ {
		kernel := p.spoofKernel()
		p.resources.Set(kernel.GetKernelId(), kernel)
	}

	app.Logf("Created an initial batch of %d spoofed kernels.", numKernels)
}

// Top-level function for spoofing kernels.
func (p *SpoofedKernelProvider) spoofKernels() {
	// If we've already generated some kernels, then we'll randomly remove a few and add a few.
	if p.resources.Count() > 0 {
		app.Log("Spoofing kernels.")

		var maxAdd int

		if p.resources.Count() <= 2 {
			// If ther's 2 kernels or less, then add up to 5.
			maxAdd = 5
		} else {
			maxAdd = int(math.Ceil((0.25 * float64(p.resources.Count())))) // Add and remove up to 25% of the existing number of the spoofed kernels.
		}

		maxDelete := int(math.Ceil((0.50 * float64(p.resources.Count())))) // Add and remove up to 50% of the existing number of the spoofed kernels.
		numToDelete := rand.Intn(int(math.Max(2, float64(maxDelete+1))))   // Delete UP TO this many.
		numToAdd := rand.Intn(int(math.Max(2, float64(maxAdd+1))))

		app.Logf("Adding %d new kernel(s) and removing up to %d existing kernel(s).", numToAdd, numToDelete)

		if numToDelete > 0 {
			currentKernels := p.Resources()
			toDelete := make([]string, 0, numToDelete)

			for i := 0; i < numToDelete; i++ {
				// We may select the same victim multiple times. It will only be deleted once, of course.
				victimIdx := rand.Intn(len(currentKernels))
				toDelete = append(toDelete, currentKernels[victimIdx].GetKernelId())
			}

			numDeleted := 0
			// Delete the victims.
			for _, id := range toDelete {
				// Make sure we didn't already delete this one.
				if _, ok := p.resources.Get(id); ok {
					p.resources.Remove(id)
					numDeleted++
				}
			}

			app.Logf("Removed %d kernel(s).", numDeleted)
		}

		for i := 0; i < numToAdd; i++ {
			kernel := p.spoofKernel()
			p.resources.Set(kernel.GetKernelId(), kernel)
		}

		app.Logf("There are now %d kernel(s).", p.resources.Count())
	} else {
		app.Log("Spoofing kernels for the first time.")
		p.spoofInitialKernels()
	}
}

// Actually refresh the kernels.
// This must not be called from the UI goroutine.
// This MUST be called with the p.refreshKernelMutex held.
// It is the caller's responsibility to release the lock afterwards.
func (p *SpoofedKernelProvider) RefreshResources() {
	locked := p.refreshMutex.TryLock()
	if !locked {
		// If we did not acquire the lock, then there's already an active refresh occurring. We'll just return.
		app.Log("There is already an active spoofed refresh operation being performed. Please wait for it to complete.")
		return
	}
	defer p.refreshMutex.Unlock()

	app.Log("Refreshing kernels.")

	p.spoofKernels()

	// Simulate some delay.
	delay_ms := rand.Int31n(1500)

	app.Logf("Sleeping for %d milliseconds.", delay_ms)

	time.Sleep(time.Millisecond * time.Duration(delay_ms))

	p.RefreshOccurred()
}
