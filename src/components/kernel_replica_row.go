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

	Replica      *gateway.JupyterKernelReplica
	errorHandler domain.ErrorHandler

	onExecuteButtonClicked ExecuteReplicaButtonClickedHandler
	onMigrateButtonClicked MigrateButtonClickedHandler
}

func NewKernelReplicaRow(replica *gateway.JupyterKernelReplica, migrateButtonClickedHandler MigrateButtonClickedHandler, executeReplicaButtonClickedHandler ExecuteReplicaButtonClickedHandler, errorHandler domain.ErrorHandler) *KernelReplicaRow {
	return &KernelReplicaRow{
		Replica:                replica,
		errorHandler:           errorHandler,
		onMigrateButtonClicked: migrateButtonClickedHandler,
		onExecuteButtonClicked: executeReplicaButtonClickedHandler,
	}
}

func (krr *KernelReplicaRow) Render() app.UI {
	return app.Tr().Role("row").Class("pf-v5-c-table__tr").Body(
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
			// pf-v5-c-button pf-m-link pf-m-inline
			// pf-v5-c-button pf-m-control pf-m-small
			app.Button().Class("pf-v5-c-button pf-m-link pf-m-inline").Type("button").Text("Execute").OnClick(func(ctx app.Context, e app.Event) {
				krr.onExecuteButtonClicked(ctx, e, krr.Replica)
			}, fmt.Sprintf("%p", krr.Replica)),
		),
		app.Td().Role("cell").Body(
			// pf-v5-c-button pf-m-link pf-m-inline
			// pf-v5-c-button pf-m-control pf-m-small
			app.Button().Class("pf-v5-c-button pf-m-link pf-m-inline").Type("button").Text("Migrate").OnClick(func(ctx app.Context, e app.Event) {
				krr.onMigrateButtonClicked(ctx, e, krr.Replica)
			}, fmt.Sprintf("%p", krr.Replica)),
		),
	)
}
