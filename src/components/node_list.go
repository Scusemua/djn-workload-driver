package components

import (
	"fmt"
	"sort"

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
	nodeProvider   domain.NodeProvider

	Nodes       []*domain.KubernetesNode
	expanded    map[string]bool
	selectedIdx int
}

func NewNodeList(nodeProvider domain.NodeProvider, errorHandler domain.ErrorHandler, radioButtonsVisible bool, onNodeSelected func(*domain.KubernetesNode)) *NodeList {
	nodeList := &NodeList{
		id:                  fmt.Sprintf("NodeList-%s", uuid.New().String()[0:26]),
		onNodeSelected:      onNodeSelected,
		nodeProvider:        nodeProvider,
		errorHandler:        errorHandler,
		radioButtonsVisible: radioButtonsVisible,
	}

	nodes := nodeProvider.Resources()
	sort.Slice(nodes, func(i int, j int) bool {
		return nodes[i].NodeId < nodes[j].NodeId
	})
	nodeList.expanded = make(map[string]bool, len(nodes))
	nodeList.Nodes = nodes

	return nodeList
}

func (nl *NodeList) OnMount(ctx app.Context) {
	nl.nodeProvider.SubscribeToRefreshes(nl.id, nl.handleNodesRefreshed)

	go nl.nodeProvider.RefreshResources()
}

func (nl *NodeList) handleNodesRefreshed(nodes []*domain.KubernetesNode) bool {
	if !nl.Mounted() {
		app.Logf("NodeList %s (%p) is not mounted; ignoring refresh.", nl.id, nl)
		return false
	}

	app.Logf("NodeList %s (%p) is handling a nodes refresh. Number of nodes: %d.", nl.id, nl, len(nodes))
	sort.Slice(nodes, func(i int, j int) bool {
		return nodes[i].NodeId < nodes[j].NodeId
	})

	refreshedExpanded := make(map[string]bool, len(nodes))

	for _, node := range nodes {
		var expanded, ok bool

		if expanded, ok = nl.expanded[node.NodeId]; ok {
			refreshedExpanded[node.NodeId] = expanded
		} else {
			expanded = false
			refreshedExpanded[node.NodeId] = false
		}
	}

	nl.expanded = refreshedExpanded
	nl.Nodes = nodes

	nl.Update()
	return true
}

func (nl *NodeList) getMaxHeight(node_id string) string {
	if nl.expanded[node_id] {
		return "250px"
	} else {
		return "0px"
	}
}

func (nl *NodeList) Render() app.UI {
	nodes := nl.Nodes

	app.Logf("(%p) Rendering NodeList with %d node(s).", nl, len(nodes))
	return app.Div().
		Class("pf-v5-c-card pf-m-expanded").
		Body(
			app.Div().Class("pf-v5-c-card__header").Body(
				app.Div().Class("pf-v5-c-card__title").Body(
					app.H2().Class("pf-v5-c-title pf-m-2xl").Text("Active Kubernetes Nodes"),
				),
				app.Div().Class("pf-v5-c-card__actions pf-m-no-offset").Body(
					app.Button().
						Class("pf-v5-c-button pf-m-inline pf-m-secondary").
						Type("button").
						Text("Refresh Nodes").
						Style("font-size", "16px").
						Style("margin-right", "16px").
						OnClick(func(ctx app.Context, e app.Event) {
							e.StopImmediatePropagation()
							app.Log("Refreshing nodes in node list.")

							go nl.nodeProvider.RefreshResources()
						}),
				),
			),
			app.Div().Class("pf-v5-c-card__expandable-content").Body(
				app.Div().Class("pf-v5-c-data-list pf-m-compact pf-m-grid-md").
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
											app.Div().
												Class("pf-v5-c-data-list__item-control").Body(
												app.If(nl.radioButtonsVisible,
													app.Div().Class("pf-v5-c-data-list__check").
														Body(
															app.Div().Class("pf-v5-c-radio").Body(
																app.Input().Class("pf-v5-c-radio__input").Type("radio").Name(fmt.Sprintf("node-list-%s-radio-buttons", nl.id)).ID(fmt.Sprintf("node-%d-radio", i)).OnInput(func(ctx app.Context, e app.Event) {
																	app.Logf("Checkbox node-%d-radio received input. Context: %v. Event: %v.", i, ctx, e)
																	nl.selectedIdx = i
																}),
															),
														)),
												app.Div().Class("pf-v5-c-data-list__toggle").Body(
													app.Button().Class("pf-v5-c-button pf-m-plain").Type("button").ID(fmt.Sprintf("expand-button-kernel-%s", nodes[i].NodeId)).Body(
														app.Div().Class("pf-v5-c-data-list__toggle-icon").Body(
															app.If(nl.expanded[nodes[i].NodeId], app.I().ID(fmt.Sprintf("expand-icon-kernel-%s", nodes[i].NodeId)).Class("fas fa-angle-down")).
																Else(app.I().ID(fmt.Sprintf("expand-icon-kernel-%s", nodes[i].NodeId)).Class("fas fa-angle-right")),
														),
													).OnClick(func(ctx app.Context, e app.Event) {
														// If there's no entry yet, then we default to false, and since we clicked the expand button, we set it to true now.
														if _, ok := nl.expanded[nodes[i].NodeId]; !ok {
															nl.expanded[nodes[i].NodeId] = true

															app.Logf("Node %s should be expanded now.", nodes[i].NodeId)
														} else {
															nl.expanded[nodes[i].NodeId] = !nl.expanded[nodes[i].NodeId]

															if nl.expanded[nodes[i].NodeId] {
																app.Logf("Node %s should be expanded now.", nodes[i].NodeId)
															} else {
																app.Logf("Node %s should be collapsed now.", nodes[i].NodeId)
															}
														}

														nl.Update()
													}, nodes[i].NodeId),
												),
											),
											app.Div().
												Class("pf-v5-c-data-list__item-content").
												Body(
													app.Div().
														Class("pf-v5-c-data-list__cell pf-m-align-left").
														Body(
															app.Div().
																Class("pf-v5-l-flex pf-m-column pf-m-space-items-none").
																Body(
																	app.Div().
																		Class("pf-v5-l-flex pf-m-column").
																		Body(
																			app.P().
																				Text("Node "+nodes[i].NodeId).
																				Style("font-weight", "bold").
																				Style("font-size", "20px"),
																		),
																),
															app.Div().Class("pf-v5-c-description-list pf-m-2-col-on-lg").
																Style("padding", "8px 0px 0px 0px").
																Body(
																	// app.Div().Class("pf-v5-c-description-list__group").Body(
																	// 	app.Div().Class("pf-v5-c-description-list__term").
																	// 		Style("margin-bottom", "-8px").Body(
																	// 		app.Span().Class("pf-v5-c-description-list__text").Body(
																	// 			app.P().Text("Status"),
																	// 		),
																	// 	),
																	// 	app.Div().Class("pf-v5-c-description-list__description").Body(
																	// 		app.Div().Class("pf-v5-c-description-list__text").Body(
																	// 			NewNodeStatusLabel(nodes[i].Status, 16),
																	// 		),
																	// 	),
																	// ),
																	app.Div().Class("pf-v5-c-description-list__group").Body(
																		app.Div().Class("pf-v5-c-description-list__term").
																			Style("margin-bottom", "-8px").Body(
																			app.Span().Class("pf-v5-c-description-list__text").Body(
																				app.P().Text("IP"),
																			),
																		),
																		app.Div().Class("pf-v5-c-description-list__description").Body(
																			app.Div().Class("pf-v5-c-description-list__text").Body(
																				app.P().Text(nodes[i].IP),
																			),
																		),
																	),
																	app.Div().Class("pf-v5-c-description-list__group").Body(
																		app.Div().Class("pf-v5-c-description-list__term").
																			Style("margin-bottom", "-8px").Body(
																			app.Span().Class("pf-v5-c-description-list__text").Body(
																				app.P().Text("Age"),
																			),
																		),
																		app.Div().Class("pf-v5-c-description-list__description").Body(
																			app.Div().Class("pf-v5-c-description-list__text").Body(
																				app.P().Text(nodes[i].Age.String()),
																			),
																		),
																	),
																),
															app.Div().
																Class("pf-v5-l-flex pf-m-wrap").
																Style("padding", "8px 0px 0px 0px").
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
									// Expanded content.
									app.Section().Style("max-height", nl.getMaxHeight(nodes[i].NodeId)).Class("pf-v5-c-data-list__expandable-content collapsed").ID(fmt.Sprintf("content-%s", nodes[i].NodeId)).Body( // .Hidden(!nl.expanded[kernel_id])
										app.Div().Class("pf-v5-c-data-list__expandable-content-body").Body(
											app.Table().Class("pf-v5-c-table pf-m-compact pf-m-grid-lg").Body(
												app.THead().Body(
													app.Tr().Role("row").Class("pf-v5-c-table__tr").Body(
														app.Th().Class("pf-v5-c-table__th").Role("columnheader").Scope("col").Body(
															app.P().Text("Pod ID"),
														),
														app.Th().Class("pf-v5-c-table__th").Role("columnheader").Scope("col").Body(
															app.P().Text("Phase"),
														),
														app.Th().Class("pf-v5-c-table__th").Role("columnheader").Scope("col").Body(
															app.P().Text("Age"),
														),
														app.Th().Class("pf-v5-c-table__th").Role("columnheader").Scope("col").Body(
															app.P().Text("IP"),
														),
													),
												),
												app.TBody().Role("rowgroup").Body(
													app.Range(nodes[i].Pods).Slice(func(j int) app.UI {
														return app.Tr().Role("row").Class("pf-v5-c-table__tr").Body(
															app.Td().Role("cell").Body(
																app.Span().Text(nodes[i].Pods[j].PodName),
															),
															app.Td().Role("cell").Body(
																NewPodStatusLabel(nodes[i].Pods[j].PodPhase, 16),
															),
															app.Td().Role("cell").Body(
																app.Span().Text(nodes[i].Pods[j].PodAge),
															),
															app.Td().Role("cell").Body(
																app.Span().Text(nodes[i].Pods[j].PodIP),
															))
													},
													),
												),
											),
										),
									),
								)
						}),
					)))
}
