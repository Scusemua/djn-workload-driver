package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
)

const (
	DefaultGatewayAddress = "127.0.0.1:9000"
)

type MigrateButtonClickedHandler func(app.Context, app.Event, *gateway.JupyterKernelReplica)

type MainWindow struct {
	app.Compo

	GatewayAddress string

	workloadDriver domain.WorkloadDriver // The Workload Driver.
	errMsg         string                // Current error message.
	err            error                 // Current operational error.

	// Field that reports whether an app update is available. False by default.
	updateAvailable bool

	replicaToMigrate  *gateway.JupyterKernelReplica
	migarateModalOpen bool
}

func (w *MainWindow) OnAppUpdate(ctx app.Context) {
	w.updateAvailable = ctx.AppUpdateAvailable() // Reports that an app update is available.
}

func (w *MainWindow) SetWorkloadDriver(driver domain.WorkloadDriver) {
	w.workloadDriver = driver
}

func (w *MainWindow) recover() {
	w.err = nil
	w.errMsg = ""
	w.Update()
}

func (w *MainWindow) HandleError(err error, errMsg string) {
	w.err = err
	w.errMsg = errMsg
	w.Update()
}

func (w *MainWindow) onUpdateClick(ctx app.Context, e app.Event) {
	// Reloads the page to display the modifications.
	ctx.Reload()
}

func (w *MainWindow) connectButtonHandler() {
	go func() {
		err := w.workloadDriver.DialGatewayGRPC(w.GatewayAddress)
		if err != nil {
			app.Log("Failed to connect via gRPC.")
			w.HandleError(err, fmt.Sprintf("Failed to connect to the Cluster Gateway gRPC server using address \"%s\"", w.GatewayAddress))
		}

		w.Update()
	}()
}

func (w *MainWindow) onMigrateButtonClicked(ctx app.Context, e app.Event, replica *gateway.JupyterKernelReplica) {
	app.Logf("User wishes to migrate replica %d of kernel %s.", replica.ReplicaId, replica.KernelId)

	w.migarateModalOpen = true
	w.replicaToMigrate = replica
}

func (w *MainWindow) onMigrateSubmit(replica *gateway.JupyterKernelReplica, targetNode *gateway.KubernetesNode) {
	err := w.workloadDriver.MigrateKernelReplica(&gateway.MigrationRequest{
		TargetReplica: &gateway.ReplicaInfo{
			KernelId:  replica.KernelId,
			ReplicaId: replica.ReplicaId,
		},
	})

	if err != nil {
		app.Logf("[ERROR] Failed to migrate replica %d of kernel %s.", replica.ReplicaId, replica.KernelId)
		w.HandleError(err, fmt.Sprintf("Could not migrate replica %d of kernel %s.", replica.ReplicaId, replica.KernelId))
		return
	} else {
		app.Logf("Successfully migrated replica %d of kernel %s!", replica.ReplicaId, replica.KernelId)
	}
}

func (w *MainWindow) handleCancel(dirty bool, clear chan struct{}, confirm func()) {
	if !dirty {
		confirm()

		clear <- struct{}{}

		w.Update()

		return
	}

	confirm()

	w.Update()
}

// The Render method is where the component appearance is defined.
func (w *MainWindow) Render() app.UI {
	// linkClass := "link heading fit unselectable"
	return app.Div().
		Class("pf-c-page").
		Body(
			app.Main().
				Class("pf-c-page__main").
				ID("driver-main").
				TabIndex(-1).
				Body(
					app.If(
						!w.workloadDriver.ConnectedToGateway(),
						app.Div().
							Class("pf-c-empty-state").
							Body(
								app.Div().
									Class("pf-c-empty-state__content").
									Style("text-align", "center").
									Body(
										app.I().
											// Class("pf-c-empty-state__icon").
											Style("content", "url(\"/web/icons/cloud-disconnected.svg\")").
											Style("color", "#203250").
											Style("font-size", "136px").
											Style("margin-bottom", "-16px").
											Aria("hidden", true),
										app.H1().
											Class("pf-c-title pf-m-lg").
											Style("font-weight", "bold").
											Style("font-size", "24pt").
											Style("margin-bottom", "-8px").
											Text("Disconnected"),
										app.Div().
											Class("pf-c-empty-state__body").
											Text("To start, please enter the IP address and port of the Cluster Gateway gRPC server and press Connect."),
										app.Div().
											Class("pf-c-form__group").
											Body(
												app.Div().
													Class("pf-c-form__group").
													Style("margin-bottom", "4px").
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
															Placeholder(DefaultGatewayAddress).
															ID("gateway-address-input").
															Required(true).
															OnInput(func(ctx app.Context, e app.Event) {
																address := ctx.JSSrc().Get("value").String()

																if address != "" {
																	w.GatewayAddress = address
																} else {
																	w.GatewayAddress = DefaultGatewayAddress
																}
															}),
													),
											),
										app.Button().
											Class("pf-c-button pf-m-primary").
											Type("button").
											Text("Connect").
											OnClick(func(ctx app.Context, e app.Event) {
												if w.GatewayAddress == "" {
													w.HandleError(domain.ErrEmptyGatewayAddr, "Cluster Gateway IP address cannot be the empty string.")
												} else {
													// w.logger.Info(fmt.Sprintf("Connect clicked! Attempting to connect to Gateway (via gRPC) at %s now...", w.GatewayAddress))
													app.Logf("Connect clicked! Attempting to connect to Gateway (via gRPC) at %s now...", w.GatewayAddress)
													go w.connectButtonHandler()
												}
											}),
									),
							)).
						Else(
							app.Div().
								Class("pf-c-empty-state").
								Body(
									app.Div().
										Class("pf-c-empty-state__content").
										Body(
											app.I().
												Style("content", "url(\"/web/icons/cloud-connected.svg\")").
												Style("color", "#203250").
												Style("font-size", "136px").
												Style("margin-bottom", "-16px").
												Aria("hidden", true),
											app.H1().
												Class("pf-c-title pf-m-lg").
												Style("font-weight", "bold").
												Style("font-size", "24pt").
												Style("margin-bottom", "-8px").
												Text("Connected"),
											app.H1().
												Class("pf-c-title pf-m-lg").
												Style("font-weight", "lighter").
												Style("font-size", "16pt").
												Style("margin-bottom", "-8px").
												Text(fmt.Sprintf("Cluster Gateway: %s", w.workloadDriver.GatewayAddress())),
											app.Br(),
											app.Div().
												Body(
													NewKernelList(w.workloadDriver, w, w.onMigrateButtonClicked),
													&NodeList{
														RadioButtonsVisible: false,
														OnNodeSelected:      func(kn *gateway.KubernetesNode) { /* Do nothing */ },
														Nodes:               nil,
													},
												),
										)),
						),
					app.If(w.updateAvailable,
						// Update available notification.
						app.Div().Class("pf-v5-c-alert pf-m-info").Aria("aria-label", "Information alert").Body(
							app.Div().Class("pf-v5-c-alert__icon").Body(
								app.I().Class("fas fa-fw fa-info-circle"),
							),
							app.P().Class("pf-v5-c-alert__title").Text("Update Available").Body(
								app.Span().Class("pf-v5-screen-reader").Text("Info alert:"),
							),
							app.Div().Class("pf-v5-c-alert__action").Body(
								app.Button().Class("pf-v5-c-button pf-m-plain").Type("button").Body(
									app.I().Class("fas fa-times"),
								),
							),
							app.Div().Class("pf-v5-c-alert__description").Body(
								app.P().Text("There is a website update available. Would you like to install the update and reload the webpage?"),
							),
							app.Button().
								Class("pf-c-button pf-m-primary update-button").
								Type("button").
								Text("Update & Refresh").
								OnClick(w.onUpdateClick),
						),
					)),
			app.Section().
				Class("").
				Body(
					app.If(w.migarateModalOpen, &MigrationModal{
						ID:          "migrate-modal",
						Icon:        "fas",
						Title:       "Migrate Kernel Replica",
						Class:       "pf-m-primary",
						Body:        "Migarte a kernel replica",
						ActionLabel: "Submit",
						Replica:     w.replicaToMigrate,
						OnSubmit:    w.onMigrateSubmit,
						OnCancel: func(dirty bool, clear chan struct{}) {
							w.handleCancel(dirty, clear, func() {
								w.migarateModalOpen = false
							})
						},
					}),
					app.If(w.err != nil, &ErrorModal{
						ID:          "error-modal",
						Icon:        "fas fa-times",
						Title:       "An Error has Occurred",
						Class:       "pf-m-danger",
						Body:        w.errMsg,
						Error:       w.err,
						ActionLabel: "Close",

						OnClose: func() {
							w.recover()
						},
						OnAction: func() {
							w.recover()
						},
					})),
		)
}
