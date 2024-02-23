package driver

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type workloadDriverImpl struct {
	rpcClient          DistributedNotebookClusterClient // gRPC client to the Cluster Gateway.
	connectedToGateway bool                             // Flag indicating whether or not we're currently connected to the Cluster Gateway.

	kernels                *cmap.ConcurrentMap[string, *JupyterKernel] // Currently-active Jupyter kernels (that we know about).
	errorHandler           domain.ErrorHandler                         // Pass errors here to be displayed to the user.
	spoofGatewayConnection bool                                        // Used for development when not actually using a real cluster.

	// logger *zap.Logger // Logger. Presently unused.
}

func NewWorkloadDriver(errorHandler domain.ErrorHandler, spoofGatewayConnection bool) *workloadDriverImpl {
	kernelMap := cmap.New[*JupyterKernel]()
	driver := &workloadDriverImpl{
		kernels:                &kernelMap,
		errorHandler:           errorHandler,
		spoofGatewayConnection: spoofGatewayConnection,
	}

	// logger, err := zap.NewDevelopment()
	// if err != nil {
	// 	panic(err)
	// }

	// driver.logger = logger

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
		d.fetchKernels()
		return nil
	}

	if gatewayAddress == "" {
		return domain.ErrEmptyGatewayAddr
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

	d.fetchKernels()

	return nil
}

func (d *workloadDriverImpl) NumKernels() int32 {
	return int32(d.kernels.Count())
}

func (d *workloadDriverImpl) Kernels() []domain.Kernel {
	kernels := make([]domain.Kernel, 0, d.NumKernels())

	for kvPair := range d.kernels.IterBuffered() {
		kernels = append(kernels, kvPair.Val)
	}

	return kernels
}

func (d *workloadDriverImpl) fetchKernels() {
	if d.spoofGatewayConnection {
		numKernels := rand.Intn(16-2) + 2

		statuses := []string{"unknown", "starting", "idle", "busy", "terminating", "restarting", "autorestarting", "dead"}

		for i := 0; i < numKernels; i++ {
			status := statuses[rand.Intn(len(statuses))]
			kernel := &JupyterKernel{
				KernelId:            uuid.New().String(),
				NumReplicas:         int32(rand.Intn(5-2) + 2),
				Status:              status,
				AggregateBusyStatus: status,
			}
			d.kernels.Set(kernel.KernelId, kernel)
		}

		return
	}

	app.Log("Fetching kernels now.")
	resp, err := d.rpcClient.ListKernels(context.TODO(), &Void{})
	if err != nil {
		d.errorHandler.HandleError(err, "Failed to fetch list of active kernels from the Cluster Gateway.")
		return
	}

	for _, kernel := range resp.Kernels {
		d.kernels.Set(kernel.KernelId, kernel)
		app.Log("Discovered active kernel! ID=%s, NumReplicas=%d, Status1=%s, Status2=%s", kernel.KernelId, kernel.NumReplicas, kernel.Status, kernel.AggregateBusyStatus)
	}
}
