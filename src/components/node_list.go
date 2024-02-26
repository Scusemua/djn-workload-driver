package components

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
)

type NodeList struct {
	app.Compo

	radioButtonsVisible bool

	id string

	onNodeSelected func(*domain.KubernetesNode)
	errorHandler   domain.ErrorHandler
	workloadDriver domain.WorkloadDriver

	Nodes       []*domain.KubernetesNode
	selectedIdx int
}

func NewNodeList(workloadDriver domain.WorkloadDriver, errorHandler domain.ErrorHandler, radioButtonsVisible bool, onNodeSelected func(*domain.KubernetesNode)) *NodeList {
	nodeList := &NodeList{
		id:             fmt.Sprintf("NodeList-%s", uuid.New().String()[0:26]),
		onNodeSelected: onNodeSelected,
		workloadDriver: workloadDriver,
		errorHandler:   errorHandler,
	}

	return nodeList
}

func (nl *NodeList) OnMount(ctx app.Context) {
	nl.workloadDriver.NodeProvider().SubscribeToRefreshes(nl.id, nl.handleNodesRefreshed)

	go nl.workloadDriver.NodeProvider().RefreshResources()
}

func (nl *NodeList) handleNodesRefreshed(nodes []*domain.KubernetesNode) {
	if !nl.Mounted() {
		app.Logf("NodeList %s (%p) is not mounted; ignoring refresh.", nl.id, nl)
		return
	}

	app.Logf("NodeList %s (%p) is handling a nodes refresh. Number of nodes: %d.", nl.id, nl, len(nodes))
	nl.Nodes = nodes
	nl.Update()
}

func (nl *NodeList) Render() app.UI {
	nodes := nl.Nodes

	return app.Div().
		Class("pf-v5-c-card pf-m-expanded").
		Body(
			app.Div().Class("pf-v5-c-card__header").Body(
				app.Div().Class("pf-v5-c-card__title").Body(
					app.H2().Class("pf-v5-c-title pf-m-xl").Text("Active Kubernetes Nodes"),
				),
			),
			app.Div().Class("pf-v5-c-card__expandable-content").Body(
				app.Div().Class("pf-v5-c-data-list pf-m-grid-md").
					Aria("label", "Node list").
					ID(keyListID).
					Body(
						app.Range(nodes).Slice(func(i int) app.UI {
							return app.Li().
								Class("pf-v5-c-data-list__item").
								Body(
									app.Div().
										Class("pf-v5-c-data-list__item-row").
										Body(
											// app.Div().Class("").Style("padding", "4px 4px").Body(
											// 	app.Button().Class("pf-v5-c-button pf-m-secondary").Type("button").
											// 		Style("margin-right", "4px").
											// 		OnClick(func(ctx app.Context, e app.Event) {
											// 			nl.selectedIdx = i
											// 			nl.onNodeSelected(nl.Nodes[i])
											// 			nl.Update()
											// 		}),
											// 	app.If(nl.radioButtonsVisible, app.Button().Class("pf-v5-c-button pf-m-secondary").Type("button").Text("Clear Selection").
											// 		Style("margin-left", "4px").
											// 		OnClick(func(ctx app.Context, e app.Event) {
											// 			nl.selectedIdx = -1
											// 			nl.onNodeSelected(nil)
											// 			nl.Update()
											// 		})),
											// ),
											app.If(nl.radioButtonsVisible, app.Div().
												Class("pf-v5-c-data-list__item-control").
												Body(
													app.Div().Class("pf-v5-c-radio").Body(
														app.Input().Class("pf-v5-c-radio__input").Type("radio").Name(fmt.Sprintf("node-%d-radio", i)).OnInput(func(ctx app.Context, e app.Event) {
															app.Logf("Checkbox node-%d-radio received input. Context: %v. Event: %v.", i, ctx, e)
															nl.selectedIdx = i
														}),
													),
												)),
											app.Div().
												Class("pf-v5-c-data-list__item-content").
												Body(
													app.Div().
														Class("pf-v5-c-data-list__cell pf-m-align-left").
														Body(
															app.Div().
																Class("pf-l-flex pf-m-column pf-m-space-items-none").
																Body(
																	app.Div().
																		Class("pf-l-flex pf-m-column").
																		Body(
																			app.P().
																				Text("Node "+nodes[i].NodeId).
																				Style("font-weight", "bold").
																				Style("font-size", "16px"),
																		),
																),
															app.Div().
																Class("pf-l-flex pf-m-wrap").
																Body(
																	&ResourceLabel{
																		ResourceName: "CPU",
																		Allocated:    nodes[i].AllocatedCPU,
																		Capacity:     nodes[i].CapacityCPU,
																		FontSize:     16,
																		Class:        "fas fa-microchip",
																		UseClass:     true,
																	},
																	&ResourceLabel{
																		ResourceName: "Memory",
																		Allocated:    nodes[i].AllocatedMemory,
																		Capacity:     nodes[i].CapacityMemory,
																		FontSize:     16,
																		Class:        "fas fa-memory",
																		UseClass:     true,
																	},
																	&ResourceLabel{
																		ResourceName: "GPU",
																		Allocated:    nodes[i].AllocatedGPUs,
																		Capacity:     nodes[i].CapacityGPUs,
																		FontSize:     16,
																		Content:      "/web/icons/gpu-icon.svg",
																		UseClass:     false,
																	},
																)),
												),
										),
								)
						}),
					)))
}
