package components

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
)

const (
	keyListID = "kernel-list"
)

type KernelList struct {
	app.Compo

	Kernels map[string]*gateway.DistributedJupyterKernel

	id             string
	workloadDriver domain.WorkloadDriver
	errorHandler   domain.ErrorHandler
	expanded       map[string]bool
	selected       map[string]bool

	onMigrateButtonClicked MigrateButtonClickedHandler
}

func NewKernelList(workloadDriver domain.WorkloadDriver, errorHandler domain.ErrorHandler, onMigrateButtonClicked MigrateButtonClickedHandler) *KernelList {
	kl := &KernelList{
		id:                     fmt.Sprintf("KernelList-%s", uuid.New().String()[0:26]),
		workloadDriver:         workloadDriver,
		errorHandler:           errorHandler,
		onMigrateButtonClicked: onMigrateButtonClicked,
		// expanded:               make(map[string]bool),
		selected: make(map[string]bool),
	}

	kl.recreateState(workloadDriver.KernelsSlice())

	// app.Logf("Created new KL: %s. Number of kernels: %d.", kl.id, len(kl.Kernels))

	return kl
}

func (kl *KernelList) recreateState(kernels []*gateway.DistributedJupyterKernel) {
	// Sort by name first.
	sort.Slice(kernels, func(i, j int) bool {
		return kernels[i].KernelId < kernels[j].KernelId
	})

	refreshedSelected := make(map[string]bool, len(kernels))
	refreshedExpanded := make(map[string]bool, len(kernels))
	refreshedKernels := make(map[string]*gateway.DistributedJupyterKernel, len(kernels))

	for _, kernel := range kernels {
		var selected, expanded, ok bool
		if selected, ok = kl.selected[kernel.KernelId]; ok {
			refreshedSelected[kernel.KernelId] = selected
		} else {
			selected = false
			refreshedSelected[kernel.KernelId] = false
		}

		if expanded, ok = kl.expanded[kernel.KernelId]; ok {
			refreshedExpanded[kernel.KernelId] = expanded
		} else {
			expanded = false
			refreshedExpanded[kernel.KernelId] = false
		}

		refreshedKernels[kernel.KernelId] = kernel

		// If this section is not expanded, then make sure the icon associated with it is pointing to the right.
		// if !expanded {
		// 	jsObject := app.Window().GetElementByID(fmt.Sprintf("expand-icon-kernel-%s", kernel.KernelId)).JSValue()

		// 	// If the icon exists already, then set it to point right.
		// 	// If it doesn't exist, then just skip. It hasn't been rendered yet; the current kernel is brand new.
		// 	if jsObject != nil && !jsObject.IsNull() {
		// 		classListJS := jsObject.Get("classList")
		// 		classListJS.Call("remove", "fa-angle-down")
		// 		classListJS.Call("add", "fa-angle-right")
		// 	}
		// }
	}
	// Assign at the end so we can use existing values in 'expanded' to set the new values of 'expanded'.
	// Like, any already-expanded entries in the list should remain expanded after we add the refreshed kernels.
	kl.expanded = refreshedExpanded
	kl.selected = refreshedSelected
	kl.Kernels = refreshedKernels
}

func (kl *KernelList) handleKernelsRefresh(kernels []*gateway.DistributedJupyterKernel) {
	if !kl.Mounted() {
		app.Logf("KernelList %s (%p) is not mounted; ignoring refresh.", kl.id, kl)
		return
	}

	app.Logf("KernelList %s (%p) is handling a kernel refresh. Number of kernels: %d.", kl.id, kl, len(kernels))
	kl.recreateState(kernels)
	kl.Update()
}

func (kl *KernelList) OnMount(ctx app.Context) {
	kl.workloadDriver.SubscribeToRefreshes(kl.id, kl.handleKernelsRefresh)

	go kl.workloadDriver.RefreshKernels()
}

func (kl *KernelList) OnDismount(ctx app.Context) {
	kl.workloadDriver.UnsubscribeFromRefreshes(kl.id)
}

func (kl *KernelList) Render() app.UI {
	// We're gonna use this a lot here.
	kernels := kl.Kernels

	// app.Logf("\n[%p] Rendering KernelList with %d kernels.", kl, len(kernels))

	return app.Div().
		Body(
			app.Div().
				Style("padding", "4px 4px 2px 2px").
				Style("margin-bottom", "8px").
				Body(
					app.Button().
						Class("pf-c-button pf-m-primary").
						Type("button").
						Text("Refresh Kernels").
						Style("font-size", "16px").
						Style("margin-right", "16px").
						OnClick(func(ctx app.Context, e app.Event) {
							e.StopImmediatePropagation()
							app.Log("Refreshing kernels in kernel list.")

							go kl.workloadDriver.RefreshKernels()
						}),
					app.Button().
						Class("pf-c-button pf-m-primary pf-m-danger").
						Type("button").
						Text("Terminate Selected Kernels").
						Style("font-size", "16px").
						OnClick(func(ctx app.Context, e app.Event) {
							e.StopImmediatePropagation()

							kernelsToTerminate := make([]*gateway.DistributedJupyterKernel, 0, len(kernels))

							for kernel_id, selected := range kl.selected {
								if selected {
									app.Logf("Kernel %s is selected. Will be terminating it.", kernel_id)
									kernelsToTerminate = append(kernelsToTerminate, kernels[kernel_id])
								}
							}

							app.Logf("Terminating %d kernels now.", len(kernelsToTerminate))
						}),
				),
			app.Div().Class("pf-c-data-list pf-m-grid-md").
				Aria("label", "Kernel list").
				ID(keyListID).
				Body(
					app.Range(kernels).Map(func(kernel_id string) app.UI {
						return app.Li().
							Class("pf-c-data-list__item").
							Body(
								app.Div().
									Class("pf-c-data-list__item-row").
									Body(
										app.Div().
											Class("pf-c-data-list__item-control").
											Body(
												app.Div().Class("pf-c-data-list__toggle").Body(
													app.Button().Class("pf-c-button pf-m-plain").Type("button").ID(fmt.Sprintf("expand-button-kernel-%s", kernel_id)).Body(
														app.Div().Class("pf-c-data-list__toggle-icon").Body(
															app.If(kl.expanded[kernel_id], app.I().ID(fmt.Sprintf("expand-icon-kernel-%s", kernel_id)).Class("fas fa-angle-down")).
																Else(app.I().ID(fmt.Sprintf("expand-icon-kernel-%s", kernel_id)).Class("fas fa-angle-right")),
														),
													).OnClick(func(ctx app.Context, e app.Event) {
														// If there's no entry yet, then we default to false, and since we clicked the expand button, we set it to true now.
														if _, ok := kl.expanded[kernel_id]; !ok {
															kl.expanded[kernel_id] = true

															app.Logf("Kernel %s should be expanded now.", kernel_id)
														} else {
															kl.expanded[kernel_id] = !kl.expanded[kernel_id]

															if kl.expanded[kernel_id] {
																app.Logf("Kernel %s should be expanded now.", kernel_id)
															} else {
																app.Logf("Kernel %s should be collapsed now.", kernel_id)
															}
														}

														// jsObject := app.Window().GetElementByID(fmt.Sprintf("expand-icon-kernel-%s", kernel_id)).JSValue()
														// if jsObject != nil && !jsObject.IsNull() {
														// 	classListJS := jsObject.Get("classList")

														// 	if kl.expanded[kernel_id] {
														// 		// It's expanded.
														// 		// Change the icon to be pointing down instead of to the right.
														// 		classListJS.Call("remove", "fa-angle-right")
														// 		classListJS.Call("add", "fa-angle-down")
														// 	} else {
														// 		// It's collapsed.
														// 		// Change the icon to be pointing to the right instead of down.
														// 		classListJS.Call("remove", "fa-angle-down")
														// 		classListJS.Call("add", "fa-angle-right")
														// 	}
														// }

														kl.Update()
													}, kernel_id),
												),
												app.Div().Class("pf-c-data-list__check").Body(
													app.Input().Type("checkbox").Name(fmt.Sprintf("check-expandable-kernel-%s", kernel_id)).OnInput(func(ctx app.Context, e app.Event) {
														kl.selected[kernel_id] = !kl.selected[kernel_id]

														if kl.selected[kernel_id] {
															app.Logf("Kernel %s should be selected now.", kernel_id)
														} else {
															app.Logf("Kernel %s should be deselected now.", kernel_id)
														}
													}),
												),
											),
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
																			Text("Kernel "+kernel_id).
																			Style("font-weight", "bold").
																			Style("font-size", "16px"),
																	),
															),
														app.Div().
															Class("pf-l-flex pf-m-wrap").
															Body(
																NewKernelReplicasLabel(kernels[kernel_id].GetNumReplicas(), 16),
																NewKernelStatusLabel(kernels[kernel_id].GetStatus(), 16)),
													),
												app.Div().
													Class("pf-c-data-list__cell pf-m-align-right pf-m-no-fill").
													Body(
														app.Button().
															Class("pf-c-button pf-m-secondary pf-m-danger").
															Type("button").
															Text("Terminate").
															Style("font-size", "16px"),
													),
											),
									),
								// Expanded content.
								app.Section().Class("pf-c-data-list__expandable-content").ID(fmt.Sprintf("content-%s", kernel_id)).Hidden(!kl.expanded[kernel_id]).Body(
									app.Div().Class("pf-c-data-list__expandable-content-body pf-m-no-padding").Body(
										app.Table().Class("pf-c-table pf-m-compact pf-m-grid-lg pf-m-no-border-rows").Body(
											app.THead().Body(
												app.Tr().Role("row").Body(
													// app.Td().Class("pf-c-table__check").Role("cell").Body(
													// 	app.Input().Type("checkbox").Name("check-all"),
													// ),
													app.Th().Role("columnheader").Scope("col").Body(
														app.P().Text("Replica"),
													),
													app.Th().Role("columnheader").Scope("col").Body(
														app.P().Text("Pod"),
													),
													app.Th().Role("columnheader").Scope("col").Body(
														app.P().Text("Node"),
													),
													app.Td(),
												),
											),
											app.TBody().Role("rowgroup").Body(
												app.Range(kernels[kernel_id].GetReplicas()).Slice(func(j int) app.UI {
													return NewKernelReplicaRow(kernels[kernel_id].GetReplicas()[j], kl.onMigrateButtonClicked, kl.workloadDriver, kl.errorHandler)
												},
												),
											),
										),
									),
								),
							)
					}),
				))
}
