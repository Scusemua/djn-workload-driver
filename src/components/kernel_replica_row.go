package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
)

// When expanding a specific kernel in a kernel list, there is a row for each of its replicas.
type KernelReplicaRow struct {
	app.Compo

	workloadDriver domain.WorkloadDriver
	Replica        *gateway.JupyterKernelReplica
	errorHandler   domain.ErrorHandler

	onMigrateButtonClickedHandler MigrateButtonClickedHandler
}

func NewKernelReplicaRow(replica *gateway.JupyterKernelReplica, onMigrateButtonClickedHandler MigrateButtonClickedHandler, workloadDriver domain.WorkloadDriver, errorHandler domain.ErrorHandler) *KernelReplicaRow {
	return &KernelReplicaRow{
		Replica:                       replica,
		workloadDriver:                workloadDriver,
		errorHandler:                  errorHandler,
		onMigrateButtonClickedHandler: onMigrateButtonClickedHandler,
	}
}

func (krr *KernelReplicaRow) Render() app.UI {
	return app.Tr().Role("row").Body(
		app.Td().Role("cell").Body(
			app.Span().Text(krr.Replica.GetReplicaId()),
		),
		app.Td().Role("cell").Body(
			app.Span().Text(krr.Replica.GetPodId()),
		),
		app.Td().Role("cell").Body(
			app.Span().Text(krr.Replica.GetNodeId()),
		),
		app.Td().Role("cell").Body(
			app.Button().Class("pf-v5-c-button pf-m-control pf-m-small").Type("button").Text("Migrate").OnClick(func(ctx app.Context, e app.Event) {
				krr.onMigrateButtonClickedHandler(ctx, e, krr.Replica)
			}, fmt.Sprintf("%p", krr.Replica)),
		),
	)
}
