package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/driver"
	"github.com/scusemua/djn-workload-driver/m/v2/src/gateway"
)

// When expanding a specific kernel in a kernel list, there is a row for each of its replicas.
type KernelReplicaRow struct {
	app.Compo

	workloadDriver driver.WorkloadDriver
	replica        *gateway.JupyterKernelReplica
	parent         *KernelList
	errorHandler   driver.ErrorHandler
}

func NewKernelReplicaRow(replica *gateway.JupyterKernelReplica, parent *KernelList, workloadDriver driver.WorkloadDriver, errorHandler driver.ErrorHandler) *KernelReplicaRow {
	return &KernelReplicaRow{
		replica:        replica,
		workloadDriver: workloadDriver,
		errorHandler:   errorHandler,
		parent:         parent,
	}
}

func (krr *KernelReplicaRow) Render() app.UI {
	return app.Tr().Role("row").Body(
		app.Td().Role("cell").Body(
			app.Span().Text(krr.replica.GetReplicaId()),
		),
		app.Td().Role("cell").Body(
			app.Span().Text(krr.replica.GetPodId()),
		),
		app.Td().Role("cell").Body(
			app.Span().Text(krr.replica.GetNodeId()),
		),
		app.Td().Role("cell").Body(
			app.Button().Class("pf-v5-c-button pf-m-control pf-m-small").Type("button").Text("Migrate").OnClick(func(ctx app.Context, e app.Event) {
				app.Logf("User wishes to migrate replica %d of kernel %s.", krr.replica.ReplicaId, krr.replica.KernelId)

				err := krr.workloadDriver.MigrateKernelReplica(&gateway.MigrationRequest{
					TargetReplica: &gateway.ReplicaInfo{
						KernelId:  krr.replica.KernelId,
						ReplicaId: krr.replica.ReplicaId,
					},
				})

				if err != nil {
					app.Logf("[ERROR] Failed to migrate replica %d of kernel %s.", krr.replica.ReplicaId, krr.replica.KernelId)
					krr.errorHandler.HandleError(err, fmt.Sprintf("Could not migrate replica %d of kernel %s.", krr.replica.ReplicaId, krr.replica.KernelId))
					return
				} else {
					app.Logf("Successfully migrated replica %d of kernel %s!", krr.replica.ReplicaId, krr.replica.KernelId)
				}

				krr.parent.Update()
			}),
		),
	)
}
