package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
)

type MigrationModal struct {
	app.Compo

	ID          string // HTML ID of the modal; must be unique across the page
	Icon        string // Class of the icon to use to the left of the title; may be empty
	Title       string // Title of the modal
	Class       string // Class to be applied to the modal's outmost component
	Body        string // Body text of the modal
	ActionLabel string // Text to display on the modal's primary action

	OnCancel func(dirty bool, clear chan struct{})                        // Handler for when we cancel the migration operation.
	OnClose  func()                                                       // Handler to call when closing/cancelling the modal
	OnSubmit func(*gateway.JupyterKernelReplica, *gateway.KubernetesNode) // Handler to call when triggering the modal's primary action

	// The replica we're migrating.
	Replica *gateway.JupyterKernelReplica
	Nodes   []*gateway.KubernetesNode

	targetNode *gateway.KubernetesNode

	dirty bool
}

func (c *MigrationModal) OnNodeSelected(selectedNode *gateway.KubernetesNode) {
	c.targetNode = selectedNode
}

func (c *MigrationModal) Render() app.UI {
	modal_id := fmt.Sprintf("migrate-kernel-%s-%d-modal", c.Replica.KernelId, c.Replica.ReplicaId)
	return &Modal{
		ID:    modal_id,
		Title: fmt.Sprintf("Migrate kernel-%s-%d", c.Replica.KernelId, c.Replica.ReplicaId),
		Body: []app.UI{
			app.Form().
				Class("pf-c-form").
				ID(modal_id).
				OnSubmit(func(ctx app.Context, e app.Event) {
					e.PreventDefault()

					c.OnSubmit(c.Replica, c.targetNode)

					c.clear()
				}).Body(
				app.Div().Class("pf-c-form__group").Body(
					&NodeList{
						RadioButtonsVisible: true,
						Nodes:               c.Nodes,
						OnNodeSelected:      c.OnNodeSelected,
						selectedIdx:         -1,
					},
				),
			),
		},
		Footer: []app.UI{
			app.Button().
				Class("pf-c-button pf-m-primary").
				Type("submit").
				Form("encrypt-and-sign-form").
				Text("Migrate"),
			app.Button().
				Class("pf-c-button pf-m-link").
				Type("button").
				Text("Cancel").
				OnClick(func(ctx app.Context, e app.Event) {
					handleCancel(c.clear, c.dirty, c.OnCancel)
				}),
		},
		OnClose: func() {
			handleCancel(c.clear, c.dirty, c.OnCancel)
		},
	}
}

func (c *MigrationModal) clear() {
	// Clear input value
	c.dirty = false
}
