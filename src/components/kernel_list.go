package components

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/driver"
	"github.com/scusemua/djn-workload-driver/m/v2/src/gateway"
)

const (
	keyListID = "kernel-list"
)

type KernelList struct {
	app.Compo

	kernels []*gateway.DistributedJupyterKernel

	id             string
	workloadDriver driver.WorkloadDriver
	errorHandler   driver.ErrorHandler
	expanded       []bool
	selected       []bool
}

func NewKernelList(workloadDriver driver.WorkloadDriver, errorHandler driver.ErrorHandler) *KernelList {
	kl := &KernelList{
		id:             fmt.Sprintf("KernelList-%s", uuid.New().String()[0:26]),
		workloadDriver: workloadDriver,
		errorHandler:   errorHandler,
	}

	return kl
}

func (kl *KernelList) handleKernelsRefresh(kernels []*gateway.DistributedJupyterKernel) {
	app.Logf("KernelList %s is handling a kernel refresh.", kl.id)

	kl.kernels = kernels
	kl.selected = make([]bool, 0, len(kernels))
	kl.expanded = make([]bool, 0, len(kernels))

	for i := 0; i < len(kernels); i++ {
		kl.selected = append(kl.selected, false)
		kl.expanded = append(kl.expanded, false)
	}
}

func (kl *KernelList) OnMount(ctx app.Context) {
	ctx.Async(func() {
		kl.workloadDriver.ManuallyRefreshKernels()

		ctx.Dispatch(func(ctx app.Context) {
			kl.handleKernelsRefresh(kl.workloadDriver.Kernels())
		})
	})
}

func (kl *KernelList) Render() app.UI {
	// We're gonna use this a lot here.
	kernels := kl.kernels

	app.Logf("Rendering KernelList with %d kernels.", len(kernels))

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

							ctx.Async(func() {
								kl.workloadDriver.ManuallyRefreshKernels()

								ctx.Dispatch(func(ctx app.Context) {
									kl.handleKernelsRefresh(kl.workloadDriver.Kernels())
								})
							})
						}),
					app.Button().
						Class("pf-c-button pf-m-primary pf-m-danger").
						Type("button").
						Text("Terminate Selected Kernels").
						Style("font-size", "16px").
						OnClick(func(ctx app.Context, e app.Event) {
							e.StopImmediatePropagation()

							kernelsToTerminate := make([]*gateway.DistributedJupyterKernel, 0, len(kl.kernels))

							for i, selected := range kl.selected {
								if selected {
									app.Logf("Kernel %s is selected. Will be terminating it.", kl.kernels[i].GetKernelId())
									kernelsToTerminate = append(kernelsToTerminate, kl.kernels[i])
								}
							}

							app.Logf("Terminating %d kernels now.", len(kernelsToTerminate))
						}),
				),
			app.Div().Class("pf-c-data-list pf-m-grid-md").
				Aria("label", "Kernel list").
				ID(keyListID).
				Body(
					app.Range(kernels).Slice(func(i int) app.UI {
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
													app.Button().Class("pf-c-button pf-m-plain").Type("button").ID(fmt.Sprintf("expand-kernel-%d", i)).Body(
														app.Div().Class("pf-c-data-list__toggle-icon").Body(
															app.I().ID(fmt.Sprintf("kernel-expand-icon-%d", i)).Class("fas fa-angle-right"),
														),
													).OnClick(func(ctx app.Context, e app.Event) {
														app.Logf("Expand button expand-kernel-%d received input. Context: %v. Event: %v.", i, ctx, e)
														kl.expanded[i] = !kl.expanded[i]

														classListJS := app.Window().GetElementByID(fmt.Sprintf("kernel-expand-icon-%d", i)).JSValue().Get("classList")
														if kl.expanded[i] {
															// It's expanded.
															// Change the icon to be pointing down instead of to the right.
															classListJS.Call("remove", "fa-angle-right")
															classListJS.Call("add", "fa-angle-down")
														} else {
															// It's collapsed.
															// Change the icon to be pointing to the right instead of down.
															classListJS.Call("remove", "fa-angle-down")
															classListJS.Call("add", "fa-angle-right")
														}

														kl.Update()
													}),
												),
												app.Div().Class("pf-c-data-list__check").Body(
													app.Input().Type("checkbox").Name(fmt.Sprintf("check-expandable-check-%d", i)).OnInput(func(ctx app.Context, e app.Event) {
														app.Logf("Checkbox check-expandable-check-%d received input. Context: %v. Event: %v.", i, ctx, e)
														kl.selected[i] = !kl.selected[i]
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
																			Text("Kernel "+kernels[i].GetKernelId()).
																			Style("font-weight", "bold").
																			Style("font-size", "16px"),
																	),
															),
														app.Div().
															Class("pf-l-flex pf-m-wrap").
															Body(
																NewKernelReplicasLabel(kernels[i].GetNumReplicas(), 16),
																NewKernelStatusLabel(kernels[i].GetStatus(), 16)),
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
								app.Section().Class("pf-c-data-list__expandable-content").ID(fmt.Sprintf("content-%d", i)).Hidden(!kl.expanded[i]).Body(
									app.Div().Class("pf-c-data-list__expandable-content-body pf-m-no-padding").Body(
										app.Table().Class("pf-c-table pf-m-compact pf-m-grid-lg pf-m-no-border-rows").Body(
											app.THead().Body(
												app.Tr().Role("row").Body(
													// app.Td().Class("pf-c-table__check").Role("cell").Body(
													// 	app.Input().Type("checkbox").Name("check-all"),
													// ),
													app.Th().Role("columnheader").Scope("col").Body(
														app.P().Text("Replica ID"),
													),
													app.Th().Role("columnheader").Scope("col").Body(
														app.P().Text("Pod ID"),
													),
													app.Th().Role("columnheader").Scope("col").Body(
														app.P().Text("Node ID"),
													),
													app.Td(),
												),
											),
											app.TBody().Role("rowgroup").Body(
												app.Range(kernels[i].GetReplicas()).Slice(func(j int) app.UI {
													return NewKernelReplicaRow(kernels[i].GetReplicas()[j], kl, kl.workloadDriver, kl.errorHandler)
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
