package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
)

type GatewayConnectionWindow struct {
	app.Compo

	workloadDriver domain.WorkloadDriver
	errorHandler   domain.ErrorHandler

	gatewayAddress string
}

func NewGatewayConnectionWindow(workloadDriver domain.WorkloadDriver, errorHandler domain.ErrorHandler) *GatewayConnectionWindow {
	return &GatewayConnectionWindow{
		workloadDriver: workloadDriver,
		errorHandler:   errorHandler,
	}
}

func (w *GatewayConnectionWindow) connectButtonHandler() {
	err := w.workloadDriver.DialGatewayGRPC(w.gatewayAddress)
	if err != nil {
		app.Log("Failed to connect via gRPC.")
		w.errorHandler.HandleError(err, fmt.Sprintf("Failed to connect to the Cluster Gateway gRPC server using address \"%s\"", w.gatewayAddress))
	}
}

func (w *GatewayConnectionWindow) Render() app.UI {
	return app.Div().
		Class("pf-c-empty-state").
		Body(
			app.Div().
				Class("pf-c-empty-state__content").
				Body(
					app.I().
						// Class("pf-c-empty-state__icon").
						Style("content", "url(\"/web/icons/cloud-disconnected.svg\")").
						Style("color", "#203250").
						Style("font-size", "128px").
						Aria("hidden", true),
					app.H1().
						Class("pf-c-title pf-m-lg").
						Text("Disconnected"),
					app.Div().
						Class("pf-c-empty-state__body").
						Text("To start, please enter the IP address and port of the Cluster Gateway gRPC server and press Connect."),
					app.Div().
						Class("pf-c-form__group").
						Body(
							app.Div().
								Class("pf-c-form__group").
								Body(
									app.Label().
										Class("pf-c-form__label").
										For("gateway-address-input").
										Body(
											app.Span().
												Class("pf-c-form__label-text").
												Text("Gateway Address"),
											app.Span().
												Class("pf-c-form__label-required").
												Aria("hidden", true).
												Text("*"),
										),
								),
							app.Div().
								Class("pf-c-form__group-control").
								Body(
									app.Input().
										Class("pf-c-form-control").
										Type("text").
										Placeholder("0.0.0.0:9000").
										ID("gateway-address-input").
										Required(true).
										OnInput(func(ctx app.Context, e app.Event) {
											w.gatewayAddress = ctx.JSSrc().Get("value").String()
										}),
								),
						),
					app.Button().
						Class("pf-c-button pf-m-primary").
						Type("button").
						Text("Connect").
						OnClick(func(ctx app.Context, e app.Event) {
							if w.gatewayAddress == "" {
								w.errorHandler.HandleError(domain.ErrEmptyGatewayAddr, "Cluster Gateway IP address cannot be the empty string.")
							} else {
								// w.logger.Info(fmt.Sprintf("Connect clicked! Attempting to connect to Gateway (via gRPC) at %s now...", w.gatewayAddress))
								app.Logf("Connect clicked! Attempting to connect to Gateway (via gRPC) at %s now...", w.gatewayAddress)
								w.connectButtonHandler()
							}

							w.Update()
						}),
				),
		)
}
