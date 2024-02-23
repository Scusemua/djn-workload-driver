package components

import (
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/driver"
)

// When expanding a specific kernel in a kernel list, there is a row for each of its replicas.
type KernelReplicaRow struct {
	app.Compo

	replica *driver.JupyterKernelReplica
}

func NewKernelReplicaRow(replica *driver.JupyterKernelReplica) *KernelReplicaRow {
	return &KernelReplicaRow{
		replica: replica,
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
			app.Button().Class("pf-v5-c-button pf-m-control pf-m-small").Type("button").Text("Migrate"),
		),
	)
}
