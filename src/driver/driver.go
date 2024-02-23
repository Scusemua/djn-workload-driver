package driver

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	cmap "github.com/orcaman/concurrent-map/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type workloadDriverImpl struct {
	rpcClient              DistributedNotebookClusterClient                       // gRPC client to the Cluster Gateway.
	connectedToGateway     bool                                                   // Flag indicating whether or not we're currently connected to the Cluster Gateway.
	gatewayAddress         string                                                 // IP address of the Gateway.
	kernels                *cmap.ConcurrentMap[string, *DistributedJupyterKernel] // Currently-active Jupyter kernels (that we know about). Map from Kernel ID to Kernel.
	errorHandler           ErrorHandler                                           // Pass errors here to be displayed to the user.
	spoofGatewayConnection bool                                                   // Used for development when not actually using a real cluster.

	// logger *zap.Logger // Logger. Presently unused.
}

func NewWorkloadDriver(errorHandler ErrorHandler, spoofGatewayConnection bool) *workloadDriverImpl {
	kernelMap := cmap.New[*DistributedJupyterKernel]()
	driver := &workloadDriverImpl{
		kernels:                &kernelMap,
		errorHandler:           errorHandler,
		spoofGatewayConnection: spoofGatewayConnection,
	}

	// logger, err := zap.NewDevelopment()
	// if err != nil {
	// 	panic(err)
	// }

	// logger = logger

	return driver
}

func (d *workloadDriverImpl) Start() {
	// Do nothing for now.
}

func (d *workloadDriverImpl) ConnectedToGateway() bool {
	return d.connectedToGateway
}

func (d *workloadDriverImpl) DialGatewayGRPC(gatewayAddress string) error {
	if d.spoofGatewayConnection {
		app.Logf("Spoofing RPC connection to Cluster Gateway...")
		time.Sleep(time.Second * 1)
		d.connectedToGateway = true
		d.gatewayAddress = gatewayAddress
		d.RefreshKernels()
		return nil
	}

	if gatewayAddress == "" {
		return ErrEmptyGatewayAddr
	}

	// d.logger.Info(fmt.Sprintf("Attempting to dial Gateway gRPC server now. Address: %s\n", d.gatewayAddress))
	app.Logf("Attempting to dial Gateway gRPC server now. Address: %s\n", gatewayAddress)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()
	conn, err := grpc.DialContext(ctx, gatewayAddress, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		// d.logger.Error(fmt.Sprintf("Failed to dial Gateway gRPC server. Address: %s. Error: %v.\n", d.gatewayAddress, zap.Error(err)))
		app.Logf("Failed to dial Gateway gRPC server. Address: %s. Error: %v.\n", gatewayAddress, zap.Error(err))
		return err
	}

	// d.logger.Info(fmt.Sprintf("Successfully dialed Cluster Gateway at address %s.\n", d.gatewayAddress))
	app.Logf("Successfully dialed Cluster Gateway at address %s.\n", gatewayAddress)
	d.rpcClient = NewDistributedNotebookClusterClient(conn)
	d.connectedToGateway = true
	d.gatewayAddress = gatewayAddress

	d.RefreshKernels()

	return nil
}

func (d *workloadDriverImpl) MigrateKernelReplica(arg *JupyterKernelArg) error {
	return nil
}

func (d *workloadDriverImpl) GatewayAddress() string {
	return d.gatewayAddress
}

func (d *workloadDriverImpl) NumKernels() int32 {
	return int32(d.kernels.Count())
}

func (d *workloadDriverImpl) Kernels() []*DistributedJupyterKernel {
	kernels := make([]*DistributedJupyterKernel, 0, d.NumKernels())

	for kvPair := range d.kernels.IterBuffered() {
		kernel := kvPair.Val
		kernels = append(kernels, kernel)
	}

	return kernels
}

func (d *workloadDriverImpl) spoofKernel() *DistributedJupyterKernel {
	status := KernelStatuses[rand.Intn(len(KernelStatuses))]
	numReplicas := rand.Intn(5-2) + 2
	kernelId := uuid.New().String()
	// Spoof the kernel itself.
	kernel := &DistributedJupyterKernel{
		KernelId:            kernelId,
		NumReplicas:         int32(numReplicas),
		Status:              status,
		AggregateBusyStatus: status,
		Replicas:            make([]*JupyterKernelReplica, 0, numReplicas),
	}

	// Spoof the kernel's replicas.
	for j := 0; j < numReplicas; j++ {
		podId := fmt.Sprintf("kernel-%s-%s", kernelId, uuid.New().String()[0:5])
		replica := &JupyterKernelReplica{
			ReplicaId: int32(j),
			KernelId:  kernelId,
			PodId:     podId,
			NodeId:    fmt.Sprintf("Node-%d", rand.Intn(4-1)+1),
		}
		kernel.Replicas = append(kernel.Replicas, replica)
	}

	return kernel
}

func (d *workloadDriverImpl) spoofInitialKernels() {
	numKernels := rand.Intn(16-2) + 2

	for i := 0; i < numKernels; i++ {
		kernel := d.spoofKernel()
		d.kernels.Set(kernel.GetKernelId(), kernel)
	}
}

func (d *workloadDriverImpl) RefreshKernels() []*DistributedJupyterKernel {
	if d.spoofGatewayConnection {
		// If we've already generated some kernels, then we'll randomly remove a few and add a few.
		if d.kernels.Count() > 0 {
			maxChange := int(0.25 * float64(d.kernels.Count())) // Add and remove up to 25% of the existing number of kernels.
			numToDelete := rand.Intn(maxChange + 1)             // Delete UP TO this many.
			numToAdd := rand.Intn(maxChange + 1)

			app.Logf("Adding %d new kernel(s) and removing up to %d existing kernel(s).", numToAdd, numToDelete)

			if numToDelete > 0 {
				currentKernels := d.Kernels()
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
					if _, ok := d.kernels.Get(id); ok {
						d.kernels.Remove(id)
						numDeleted++
					}
				}

				app.Log("Removed %d kernel(s).", numDeleted)
			}

			for i := 0; i < numToAdd; i++ {
				kernel := d.spoofKernel()
				d.kernels.Set(kernel.GetKernelId(), kernel)
			}
		} else {
			d.spoofInitialKernels()
		}

		return d.Kernels()
	}

	app.Log("Fetching kernels now.")
	resp, err := d.rpcClient.ListKernels(context.TODO(), &Void{})
	if err != nil {
		d.errorHandler.HandleError(err, "Failed to fetch list of active kernels from the Cluster Gateway.")
		return d.Kernels()
	}

	for _, kernel := range resp.Kernels {
		d.kernels.Set(kernel.KernelId, kernel)
		app.Log("Discovered active kernel! ID=%s, NumReplicas=%d, Status1=%s, Status2=%s", kernel.KernelId, kernel.NumReplicas, kernel.Status, kernel.AggregateBusyStatus)
	}

	return d.Kernels()
}
