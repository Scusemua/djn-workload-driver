package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
)

const (
	keyListID = "kernel-list"
)

type KernelList struct {
	app.Compo

	kernelProvider domain.KernelProvider
}

func NewKernelList(kernelProvider domain.KernelProvider) *KernelList {
	return &KernelList{
		kernelProvider: kernelProvider,
	}
}

func (kl *KernelList) Render() app.UI {
	// We're gonna use this a lot here.
	kernels := kl.kernelProvider.Kernels()

	app.Logf("Rendering KernelList with %d kernels.", len(kernels))

	return app.Div().
		Class("pf-c-empty-state").
		Body(
			app.Div().
				Class("pf-c-empty-state__content").
				Body(
					app.I().
						Style("content", "url(\"/web/icons/cloud-connected.svg\")").
						Style("color", "#203250").
						Style("font-size", "136px").
						Aria("hidden", true),
					app.H1().
						Class("pf-c-title pf-m-lg").
						Text(fmt.Sprintf("Connected to Cluster Gateway at %s", kl.kernelProvider.GatewayAddress())),
					app.Div().
						Class("pf-c-data-list pf-m-grid-md").
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
																					Style("font-size", "32px"),
																			),
																	),
																app.Div().
																	Class("pf-l-flex pf-m-wrap").
																	Body(
																		NewKernelReplicas(kernels[i].GetNumReplicas()),
																		NewKernelStatus(kernels[i].GetStatus())),
															),
														app.Div().
															Class("pf-c-data-list__cell pf-m-align-right pf-m-no-fill").
															Body(
																app.Button().
																	Class("pf-c-button pf-m-secondary pf-m-danger").
																	Type("button").
																	Text("Terminate").
																	Style("font-size", "24px"),
															),
													),
											),
									)
							}),
						)))
}
