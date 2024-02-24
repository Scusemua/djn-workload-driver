package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
)

type NodeList struct {
	app.Compo

	RadioButtonsVisible bool
	OnNodeSelected      func(*gateway.KubernetesNode)

	Nodes       []*gateway.KubernetesNode
	selectedIdx int
}

func (nl *NodeList) Render() app.UI {
	nodes := nl.Nodes

	return app.Div().Class("pf-c-data-list pf-m-grid-md").
		Aria("label", "Kernel list").
		ID(keyListID).
		Body(
			app.Range(nodes).Slice(func(i int) app.UI {
				return app.Li().
					Class("pf-c-data-list__item").
					Body(
						app.Div().
							Class("pf-c-data-list__item-row").
							Body(
								app.Div().Class("").Style("padding", "4px 4px").Body(
									app.Button().Class("pf-c-button pf-m-secondary").Type("button").Text("Refresh Nodes").
										Style("margin-right", "4px").
										OnClick(func(ctx app.Context, e app.Event) {
											nl.selectedIdx = i
											nl.OnNodeSelected(nl.Nodes[i])
											nl.Update()
										}),
									app.If(nl.RadioButtonsVisible, app.Button().Class("pf-c-button pf-m-secondary").Type("button").Text("Clear Selection").
										Style("margin-left", "4px").
										OnClick(func(ctx app.Context, e app.Event) {
											nl.selectedIdx = -1
											nl.OnNodeSelected(nil)
											nl.Update()
										})),
								),
								app.If(nl.RadioButtonsVisible, app.Div().
									Class("pf-c-data-list__item-control").
									Body(
										app.Div().Class("pf-v5-c-radio").Body(
											app.Input().Class("pf-v5-c-radio__input").Type("radio").Name(fmt.Sprintf("node-%d-radio", i)).OnInput(func(ctx app.Context, e app.Event) {
												app.Logf("Checkbox node-%d-radio received input. Context: %v. Event: %v.", i, ctx, e)
												nl.selectedIdx = i
											}),
										),
									)),
								app.Div().
									Class("pf-c-data-list__item-content").
									Body(
										app.Div().
											Class("pf-c-data-list__cell pf-m-align-left").
											Body(
												app.Div().
													Class("pf-l-flex pf-m-column pf-m-space-items-none").
													Body(
														app.Div().
															Class("pf-l-flex pf-m-column").
															Body(
																app.P().
																	Text("Kernel "+nodes[i].GetNodeId()).
																	Style("font-weight", "bold").
																	Style("font-size", "16px"),
															),
													),
												app.Div().
													Class("pf-l-flex pf-m-wrap").
													Body(
														&ResourceLabel{
															ResourceName: "CPU",
															Allocated:    nodes[i].GetAllocatedCPU(),
															Allocatable:  nodes[i].GetAllocatableCPU(),
															FontSize:     16,
															Class:        "fas fa-microchip",
															UseClass:     true,
														},
														&ResourceLabel{
															ResourceName: "Memory",
															Allocated:    nodes[i].GetAllocatedCPU(),
															Allocatable:  nodes[i].GetAllocatableCPU(),
															FontSize:     16,
															Class:        "fas fa-memory",
															UseClass:     true,
														},
														&ResourceLabel{
															ResourceName: "GPU",
															Allocated:    nodes[i].GetAllocatedCPU(),
															Allocatable:  nodes[i].GetAllocatableCPU(),
															FontSize:     16,
															Content:      "/web/icnos/gpu-icon.svg",
															UseClass:     false,
														},
													)),
									),
							),
					)
			}),
		)
}
