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
		selected:               make(map[string]bool),
	}

	kl.recreateState(workloadDriver.KernelProvider().Resources())

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
	kl.workloadDriver.KernelProvider().SubscribeToRefreshes(kl.id, kl.handleKernelsRefresh)

	go kl.workloadDriver.KernelProvider().RefreshResources()
}

func (kl *KernelList) OnDismount(ctx app.Context) {
	kl.workloadDriver.KernelProvider().UnsubscribeFromRefreshes(kl.id)
}

func (kl *KernelList) Render() app.UI {
	// We're gonna use this a lot here.
	kernels := kl.Kernels

	// app.Logf("\n[%p] Rendering KernelList with %d kernels.", kl, len(kernels))

	return app.Div().
		Class("pf-v5-c-card pf-m-expanded").
		Body(
			app.Div().Class("pf-v5-c-card__header").Body(
				app.Div().Class("pf-v5-c-card__title").Body(
					app.H2().Class("pf-v5-c-title pf-m-xl").Text("Active Kernels"),
				),
			),
			app.Div().Class("pf-v5-c-card__body").Body(
				app.Div().Class("pf-v5-c-data-list pf-m-grid-md").
					Aria("label", "Kernel list").
					ID(keyListID).
					Body(
						app.Range(kernels).Map(func(kernel_id string) app.UI {
							return app.Li().
								Class("pf-v5-c-data-list__item").
								Body(
									app.Div().
										Class("pf-v5-c-data-list__item-row").
										Body(
											app.Div().
												Class("pf-v5-c-data-list__item-control").
												Body(
													app.Div().Class("pf-v5-c-data-list__toggle").Body(
														app.Button().Class("pf-v5-c-button pf-m-plain").Type("button").ID(fmt.Sprintf("expand-button-kernel-%s", kernel_id)).Body(
															app.Div().Class("pf-v5-c-data-list__toggle-icon").Body(
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

															kl.Update()
														}, kernel_id),
													),
													app.Div().Class("pf-v5-c-data-list__check").Body(
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
												Class("pf-v5-c-data-list__item-content").
												Body(
													app.Div().
														Class("pf-v5-c-data-list__cell pf-m-align-left").
														Body(
															app.Div().
																Class("pf-v5-l-flex pf-m-column pf-m-space-items-none").
																Style("padding", "8px 0px 0px 0px").
																Body(
																	app.Div().
																		Class("pf-v5-l-flex pf-m-column").
																		Body(
																			app.P().
																				Text("Kernel "+kernel_id).
																				Style("font-weight", "bold").
																				Style("font-size", "20px"),
																		),
																),
															app.Div().
																Class("pf-v5-l-flex pf-m-wrap").
																Body(
																	NewKernelReplicasLabel(kernels[kernel_id].GetNumReplicas(), 16),
																	NewKernelStatusLabel(kernels[kernel_id].GetStatus(), 16)),
														),
													app.Div().
														Class("pf-v5-c-data-list__cell pf-m-align-right pf-m-no-fill").
														Body(
															app.Button().
																Class("pf-v5-c-button pf-m-secondary pf-m-danger").
																Type("button").
																Text("Terminate").
																Style("font-size", "16px"),
														),
												),
										),
									// Expanded content.
									app.Section().Style("max-height", kl.getMaxHeight(kernel_id)).Class("pf-v5-c-data-list__expandable-content collapsed").ID(fmt.Sprintf("content-%s", kernel_id)).Body( // .Hidden(!kl.expanded[kernel_id])
										app.Div().Class("pf-v5-c-data-list__expandable-content-body").Body(
											app.Table().Class("pf-v5-c-table pf-m-compact pf-m-grid-lg").Body(
												app.THead().Body(
													app.Tr().Role("row").Class("pf-v5-c-table__tr").Body(
														// app.Td().Class("pf-v5-c-table__check").Role("cell").Body(
														// 	app.Input().Type("checkbox").Name("check-all"),
														// ),
														app.Th().Class("pf-v5-c-table__th").Role("columnheader").Scope("col").Body(
															app.P().Text("Replica ID"),
														),
														app.Th().Class("pf-v5-c-table__th").Role("columnheader").Scope("col").Body(
															app.P().Text("Pod Name"),
														),
														app.Th().Class("pf-v5-c-table__th").Role("columnheader").Scope("col").Body(
															app.P().Text("Node Name"),
														),
														app.Td().Class("pf-v5-c-table__td"),
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
					)))
}

func (kl *KernelList) getMaxHeight(kernel_id string) string {
	if kl.expanded[kernel_id] {
		return "250px"
	} else {
		return "0px"
	}
}
