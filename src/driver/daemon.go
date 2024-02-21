package driver

import (
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"go.uber.org/zap"
)

type workloadDriverImpl struct {
	app.Compo

	gatewayAddress string      // gRPC address of the Gateway. Manually entered by the user.
	logger         *zap.Logger // Logger.
}

func NewWorkloadDriver() *workloadDriverImpl {
	driver := &workloadDriverImpl{}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	driver.logger = logger

	return driver
}

// The Render method is where the component appearance is defined.
func (d *workloadDriverImpl) Render() app.UI {
	return app.Div().
		Class("pf-c-page").
		Body(
			app.Main().
				Class("pf-c-page__main").
				ID("driver-main").
				TabIndex(-1).
				Body(
					app.Section().
						Class("pf-c-page__main-section pf-m-fill").
						Body(
							app.Div().
								Class("pf-c-empty-state").
								Body(
									app.Div().
										Class("pf-c-empty-state__content").
										Body(
											app.I().
												Class("fas fa-laptop-code pf-c-empty-state__icon").
												Style("color", "#203250").
												Style("font-size", "96px").
												Aria("hidden", true),
											app.H1().
												Class("pf-c-title pf-m-lg").
												Text("No Kernels Loaded"),
											app.Div().
												Class("pf-c-empty-state__body").
												Text("To get started, please enter the address and port of the Gateway and connect."),
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
																	d.gatewayAddress = ctx.JSSrc().Get("value").String()
																}),
														),
												),
											app.Button().
												Class("pf-c-button pf-m-primary").
												Type("button").
												Text("Connect").
												OnClick(func(ctx app.Context, e app.Event) {
													app.Logf("Connect clicked! Attempting to connect to Gateway at %s now...", d.gatewayAddress)
												}),
										),
								))))
}
