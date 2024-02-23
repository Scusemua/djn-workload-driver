package driver

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/scusemua/djn-workload-driver/m/v2/src/gateway"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrRequestIgnoredCxnSpoofed = errors.New("migration operation cannot be performed as the connection to the Cluster Gateway is spoofed")
	ErrRpcDisconnected          = errors.New("cannot perform the requested RPC as we are not connected to the Cluster Gateway")
)

type workloadDriverImpl struct {
	rpcClient              gateway.ClusterGatewayClient                                   // gRPC client to the Cluster Gateway.
	connectedToGateway     bool                                                           // Flag indicating whether or not we're currently connected to the Cluster Gateway.
	gatewayAddress         string                                                         // IP address of the Gateway.
	kernels                *cmap.ConcurrentMap[string, *gateway.DistributedJupyterKernel] // Currently-active Jupyter kernels (that we know about). Map from Kernel ID to Kernel.
	errorHandler           ErrorHandler                                                   // Pass errors here to be displayed to the user.
	spoofGatewayConnection bool                                                           // Used for development when not actually using a real cluster.
	queryInterval          time.Duration                                                  // How frequently to query the Gateway for Kernel updates. TODO(Ben): Eventually, the Gateway will publish this info to us proactively.
	queryTicker            *time.Ticker                                                   // Sends ticks to update the kernels.
	quitQueryChannel       chan struct{}                                                  // Used to tell the Gateway querier (a goroutine) to stop querying.
	rpcCallTimeout         time.Duration                                                  // Timeout for individual RPC calls.
	lastKernelRefresh      time.Time                                                      // The last time we refreshed the kernels.
	refreshCallbacks       *cmap.ConcurrentMap[string, KernelRefreshCallback]             // Registered callbacks for when we refresh the kernel list.
	lastKernelRefreshMutex sync.Mutex                                                     // Sychronizes access to the lastKernelRefresh variable.                                                       // 1 indicates that we're currently updating; 0 indicates that we're not.
	refreshMutex           sync.Mutex                                                     // Sychrnoize refreshes.
}

func NewWorkloadDriver(errorHandler ErrorHandler, spoofGatewayConnection bool, queryInterval time.Duration) *workloadDriverImpl {
	kernelMap := cmap.New[*gateway.DistributedJupyterKernel]()
	refreshCallbacks := cmap.New[KernelRefreshCallback]()
	driver := &workloadDriverImpl{
		kernels:                &kernelMap,
		errorHandler:           errorHandler,
		spoofGatewayConnection: spoofGatewayConnection,
		queryInterval:          queryInterval,
		queryTicker:            time.NewTicker(queryInterval),
		refreshCallbacks:       &refreshCallbacks,
		quitQueryChannel:       make(chan struct{}),
		// updatingKernels:        0,
	}

	// logger, err := zap.NewDevelopment()
	// if err != nil {
	// 	panic(err)
	// }

	// logger = logger

	return driver
}

// Periodically query the Gateway for an update of the current active kernels.
// This should be called from its own goroutine.
//
// TODO(Ben): Eventually, the Gateway will publish this info to us proactively.
func (d *workloadDriverImpl) queryKernels() {
	for {
		select {
		case <-d.queryTicker.C:
			d.lastKernelRefreshMutex.Lock()
			// If we've manually refreshed the kernels since the last query interval, then we'll just wait to do our refresh.
			if time.Since(d.lastKernelRefresh) < d.queryInterval {
				d.lastKernelRefreshMutex.Unlock()
				continue
			}
			d.lastKernelRefreshMutex.Unlock()

			if !d.connectedToGateway {
				app.Log("[WARNING] Disconnected from Gateway; cannot query for kernel updates.")
				return
			}

			locked := d.refreshMutex.TryLock()
			if !locked {
				// If we did not acquire the lock, then there's already an active refresh occurring. We'll just return.
				continue
			}
			d.refreshKernels()
			d.refreshMutex.Unlock()
		case <-d.quitQueryChannel:
			app.Log("Ceasing queries to Gateway.")
			return
		}
	}
}

func (d *workloadDriverImpl) Start() {
	// If we're not spoofing the Gateway connection, then periodically query the Gateway for an update of the current active kernels.
	if !d.spoofGatewayConnection {
		app.Log("Starting Gateway Querier now.")
		go d.queryKernels()
	}
}

func (d *workloadDriverImpl) ConnectedToGateway() bool {
	return d.connectedToGateway
}

// This should NOT be called from the UI goroutine.
func (d *workloadDriverImpl) DialGatewayGRPC(gatewayAddress string) error {
	if d.spoofGatewayConnection {
		app.Logf("Spoofing RPC connection to Cluster Gateway...")
		time.Sleep(time.Second * 1)
		d.connectedToGateway = true
		d.gatewayAddress = gatewayAddress
	} else if gatewayAddress == "" {
		return ErrEmptyGatewayAddr
	} else {
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
		d.rpcClient = gateway.NewClusterGatewayClient(conn)
		d.connectedToGateway = true
		d.gatewayAddress = gatewayAddress
	}

	return nil
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

func (d *workloadDriverImpl) GatewayAddress() string {
	return d.gatewayAddress
}

func (d *workloadDriverImpl) NumKernels() int32 {
	return int32(d.kernels.Count())
}

func (d *workloadDriverImpl) Kernels() []*gateway.DistributedJupyterKernel {
	kernels := make([]*gateway.DistributedJupyterKernel, 0, d.NumKernels())

	for kvPair := range d.kernels.IterBuffered() {
		kernels = append(kernels, kvPair.Val)
	}

	return kernels
}

// Create an individual spoofed/fake kernel.
func (d *workloadDriverImpl) spoofKernel() *gateway.DistributedJupyterKernel {
	status := KernelStatuses[rand.Intn(len(KernelStatuses))]
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
func (d *workloadDriverImpl) spoofInitialKernels() {
	numKernels := rand.Intn(16-2) + 2

	for i := 0; i < numKernels; i++ {
		kernel := d.spoofKernel()
		d.kernels.Set(kernel.GetKernelId(), kernel)
	}

	app.Logf("Created an initial batch of %d spoofed kernels.", numKernels)
}

// Top-level function for spoofing kernels.
func (d *workloadDriverImpl) spoofKernels() {
	// If we've already generated some kernels, then we'll randomly remove a few and add a few.
	if d.kernels.Count() > 0 {
		app.Log("Spoofing kernels.")

		var maxAdd int

		if d.kernels.Count() <= 2 {
			// If ther's 2 kernels or less, then add up to 5.
			maxAdd = 5
		} else {
			maxAdd = int(math.Ceil((0.25 * float64(d.kernels.Count())))) // Add and remove up to 25% of the existing number of the spoofed kernels.
		}

		maxDelete := int(math.Ceil((0.50 * float64(d.kernels.Count())))) // Add and remove up to 50% of the existing number of the spoofed kernels.
		numToDelete := rand.Intn(int(math.Max(2, float64(maxDelete+1)))) // Delete UP TO this many.
		numToAdd := rand.Intn(int(math.Max(2, float64(maxAdd+1))))

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

			app.Logf("Removed %d kernel(s).", numDeleted)
		}

		for i := 0; i < numToAdd; i++ {
			kernel := d.spoofKernel()
			d.kernels.Set(kernel.GetKernelId(), kernel)
		}
	} else {
		app.Log("Spoofing kernels for the first time.")
		d.spoofInitialKernels()
	}
}

// Actually refresh the kernels.
// This must not be called from the UI goroutine.
// This MUST be called with the d.refreshMutex held.
// It is the caller's responsibility to release the lock afterwards.
func (d *workloadDriverImpl) refreshKernels() {
	app.Log("Refreshing kernels.")

	if d.spoofGatewayConnection {
		d.spoofKernels()

		// Simulate some delay.
		delay_ms := rand.Int31n(1500)

		app.Logf("Sleeping for %d milliseconds.", delay_ms)

		time.Sleep(time.Millisecond * time.Duration(delay_ms))
	} else {
		app.Log("Fetching kernels now.")
		resp, err := d.rpcClient.ListKernels(context.TODO(), &gateway.Void{})
		if err != nil {
			d.errorHandler.HandleError(err, "Failed to fetch list of active kernels from the Cluster Gateway.")
		}

		// Clear the current kernels.
		d.kernels.Clear()
		for _, kernel := range resp.Kernels {
			d.kernels.Set(kernel.KernelId, kernel)
			app.Log("Discovered active kernel! ID=%s, NumReplicas=%d, Status1=%s, Status2=%s", kernel.KernelId, kernel.NumReplicas, kernel.Status, kernel.AggregateBusyStatus)
		}
	}

	d.lastKernelRefresh = time.Now()
}

// Manually/explicitly refresh the set of active kernels from the Cluster Gateway.
// This does not notify anybody that the kernels have been refreshed.
// This should not be called on the UI goroutine.
func (d *workloadDriverImpl) ManuallyRefreshKernels() {
	app.Log("Manually refreshing kernels now.")

	locked := d.refreshMutex.TryLock()
	if !locked {
		// If we did not acquire the lock, then there's already an active refresh occurring. We'll just return.
		app.Log("There is already an active refresh operation being performed. Please wait for it to complete.")
		return
	}

	d.refreshKernels()
	d.refreshMutex.Unlock()
}
