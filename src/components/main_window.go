package components

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	gateway "github.com/scusemua/djn-workload-driver/m/v2/api/proto"
	"github.com/scusemua/djn-workload-driver/m/v2/src/config"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
	"github.com/scusemua/djn-workload-driver/m/v2/src/driver"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const (
	DefaultGatewayAddress = "127.0.0.1:9000"
)

type MigrateButtonClickedHandler func(app.Context, app.Event, *gateway.JupyterKernelReplica)

type MainWindow struct {
	app.Compo

	GatewayAddress string

	Alerts *orderedmap.OrderedMap[string, *Alert]

	ConfigurationReceived bool

	WorkloadDriver domain.WorkloadDriver // The Workload Driver.
	errMsg         string                // Current error message.
	err            error                 // Current operational error.

	// Field that reports whether an app update is available. False by default.
	updateAvailable bool

	replicaToMigrate  *gateway.JupyterKernelReplica
	migarateModalOpen bool
}

func NewMainWindow(gatewayAddress string) *MainWindow {
	window := &MainWindow{
		GatewayAddress:        gatewayAddress,
		Alerts:                orderedmap.NewOrderedMap[string, *Alert](),
		ConfigurationReceived: false,
		WorkloadDriver:        nil,
	}

	return window
}

// Issue a websockets request to the backend to retrieve the configuration.
func (w *MainWindow) getConfigFromBackend(ctx app.Context) {
	ctxConnect, cancelConnect := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelConnect()
	c, _, err := websocket.Dial(ctxConnect, "ws://localhost:9995/api/config", nil)
	if err != nil {
		app.Logf("Failed to connect to backend while trying to get config: %v", err)
		w.HandleError(err, "Failed to fetch config from backend. Could not connect to the backend.")
		return
	}
	defer c.CloseNow()

	msg := map[string]interface{}{
		"op": "request-config",
	}

	ctxWrite, cancelWrite := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelWrite()
	err = wsjson.Write(ctxWrite, c, msg)
	if err != nil {
		ctx.Dispatch(func(ctx app.Context) {
			w.HandleError(err, "Failed to fetch list of active nodes from the Cluster Gateway.")
		})
		return
	}

	ctxRead, cancelRead := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelRead()
	var opts config.Configuration
	_, response, err := c.Read(ctxRead)
	c.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		app.Logf("Error encountered while reading configuration from backend: %v", err)

		ctx.Dispatch(func(ctx app.Context) {
			w.HandleError(err, "Failed to read configuration from backend.")
		})
		return
	}

	json.Unmarshal(response, &opts)
	if !opts.Valid {
		var errMessage domain.ErrorMessage
		json.Unmarshal(response, &errMessage)

		if errMessage.Valid {
			ctx.Dispatch(func(ctx app.Context) {
				app.Logf("Error encountered while reading configuration from backend: %s", errMessage.ErrorMessage)
				w.HandleError(err, "Failed to read configuration from backend.")
			})
		} else {
			// We didn't receivea valid config, but we also didn't receive a valid error message.
			panic("Unknown error while receiving config from backend.")
		}
	}

	app.Logf("Received configuration from the backend: %v", opts)

	ctx.Dispatch(func(ctx app.Context) {
		w.onConfigReceived(&opts)
	})
}

// Called when we receive the configuration from the backend.
func (w *MainWindow) onConfigReceived(opts *config.Configuration) {
	driver := driver.NewWorkloadDriver(w, opts)
	w.WorkloadDriver = driver
	w.ConfigurationReceived = true
	w.Update()
}

func (w *MainWindow) OnMount(ctx app.Context) {
	app.Log("Mounting MainWindow.")

	if app.IsServer {
		return
	}

	app.Log("Retrieving configuration from server.")

	// Get the configuration. The UI will be updated once we receive the configuration,
	// at which point we'll create the Workload Driver component.
	go w.getConfigFromBackend(ctx)
}

func (w *MainWindow) OnAppUpdate(ctx app.Context) {
	w.updateAvailable = ctx.AppUpdateAvailable() // Reports that an app update is available.

	w.addAlert(&Alert{
		ID:               uuid.New().String(),
		Name:             "Update Available",
		Class:            "pf-v5-c-alert pf-m-info",
		IconWrapperClass: "pf-v5-c-alert__icon",
		IconClass:        "fas fa-fw fa-info-circle",
		Title:            "Update Available",
		Description:      "There is a website update available.",
		ButtonText:       "Update & Refresh",
		ButtonClass:      "pf-v5-c-button pf-m-link pf-m-inline",
		ButtonOnClick:    w.onUpdateClick,
		OnClose:          w.onAlertClosed,
		HasButton:        true,
	})
}

func (w *MainWindow) onAlertClosed(alertId string, ctx app.Context, evt app.Event) {
	app.Logf("Closing alert \"%s\" now.", alertId)
	w.Alerts.Delete(alertId)
	w.Update()
}

// func (w *MainWindow) SetWorkloadDriver(driver domain.WorkloadDriver) {
// 	w.WorkloadDriver = driver
// }

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

func (w *MainWindow) onUpdateClick(alertId string, ctx app.Context, e app.Event) {
	// Reloads the page to display the modifications.
	ctx.Reload()
}

func (w *MainWindow) connectButtonHandler() {
	go func() {
		err := w.WorkloadDriver.DialGatewayGRPC(w.GatewayAddress)
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

func (w *MainWindow) onMigrateSubmit(replica *gateway.JupyterKernelReplica, targetNode *domain.KubernetesNode) {
	err := w.WorkloadDriver.MigrateKernelReplica(&gateway.MigrationRequest{
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

func (w *MainWindow) addAlert(alert *Alert) {
	app.Logf("Adding new alert: '%s'", alert.Name)
	w.Alerts.Set(alert.ID, alert)
	w.Update()
}

// app.If() always evaluates the elements to be returned, even if the condition is false.
// This functions exists to basically avoid evaluating all of the elements except when the condition is met.
func (w *MainWindow) getPreConfigReceivedUI() app.UI {
	if !w.ConfigurationReceived {
		return app.Div().Class("pf-v5-c-empty-state").
			Body(
				app.Div().
					Class("pf-v5-c-empty-state__content").
					Style("text-align", "center").
					Body(
						app.Div().
							Class("pf-v5-c-empty-state__content").
							Style("text-align", "center").
							Body(
								app.I().
									Class("pf-v5-c-empty-state__icon fas fa-spinner fa-pulse fa-spin").
									Style("color", "#203250").
									Style("font-size", "136px").
									Style("margin-bottom", "-32px").
									Aria("hidden", true),
								app.H1().
									Class("pf-v5-c-title pf-m-lg").
									Style("font-weight", "bold").
									Style("font-size", "24pt").
									Style("margin-bottom", "-8px").
									Text("Loading Configuration"),
								app.Div().
									Class("pf-v5-c-empty-state__body").
									Text("Please wait while the System Configuration is retrieved from the server."),
							)))
	} else {
		return app.Div()
	}
}

// app.If() always evaluates the elements to be returned, even if the condition is false.
// This functions exists to basically avoid evaluating all of the elements except when the condition is met.
func (w *MainWindow) getPreGatewayConnectionUI() app.UI {
	if w.ConfigurationReceived && !w.WorkloadDriver.ConnectedToGateway() {
		return app.Div().
			Class("pf-v5-c-empty-state").
			Body(
				app.Div().
					Class("pf-v5-c-empty-state__content").
					Style("text-align", "center").
					Body(
						app.I().
							// Class("pf-v5-c-empty-state__icon").
							Style("content", "url(\"/web/icons/cloud-disconnected.svg\")").
							Style("color", "#203250").
							Style("font-size", "136px").
							Style("margin-bottom", "-32px").
							Aria("hidden", true),
						app.H1().
							Class("pf-v5-c-title pf-m-lg").
							Style("font-weight", "bold").
							Style("font-size", "24pt").
							Style("margin-bottom", "-8px").
							Text("Disconnected"),
						app.Div().
							Class("pf-v5-c-empty-state__body").
							Text("To start, please enter the IP address and port of the Cluster Gateway gRPC server and press Connect."),
						app.Div().Class("pf-v5-c-form").Body(
							app.Div().
								Class("pf-v5-c-form__group").
								Style("margin-bottom", "8px").
								Body(
									app.Div().
										Class("pf-v5-c-form__group").
										Body(
											app.Label().
												Class("pf-v5-c-form__label").
												For("gateway-address-input").
												Body(
													app.Span().
														Class("pf-v5-c-form__label-text").
														Text("Gateway Address"),
													app.Span().
														Class("pf-v5-c-form__label-required").
														Aria("hidden", true).
														Text("*"),
												),
										),
									app.Div().
										Class("pf-v5-c-form__group-control").
										Body(
											app.Input().
												Class("pf-v5-c-form-control").
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
								)),
						app.Button().
							Class("pf-v5-c-button pf-m-primary").
							Type("button").
							Text("Connect").
							Style("font-size", "16px").
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
			)
	} else {
		return app.Div()
	}
}

// app.If() always evaluates the elements to be returned, even if the condition is false.
// This functions exists to basically avoid evaluating all of the elements except when the condition is met.
//
// This is the UI that is used once we've received the configuration from the backand and have connected to the Cluster Gateway.
func (w *MainWindow) getUI() app.UI {
	if w.ConfigurationReceived && w.WorkloadDriver.ConnectedToGateway() {
		return app.Div().Body(
			app.Div().
				Class("pf-v5-c-empty-state").
				Body(
					app.Div().
						Class("pf-v5-c-empty-state__content").
						Body(
							app.I().
								Style("content", "url(\"/web/icons/cloud-connected.svg\")").
								Style("color", "#203250").
								Style("font-size", "136px").
								Style("margin-bottom", "-16px").
								Aria("hidden", true),
							app.H1().
								Class("pf-v5-c-title pf-m-lg").
								Style("font-weight", "bold").
								Style("font-size", "24pt").
								Style("margin-bottom", "-8px").
								Text("Connected"),
							app.H1().
								Class("pf-v5-c-title pf-m-lg").
								Style("font-weight", "lighter").
								Style("font-size", "16pt").
								Style("margin-bottom", "-8px").
								Text(fmt.Sprintf("Cluster Gateway: %s", w.WorkloadDriver.GatewayAddress())),
							app.Br(),
							app.Div().
								Body(
									app.Div().Body(
										app.Div().
											Style("padding", "4px 4px 2px 2px").
											Style("margin-bottom", "8px").
											Body(
												app.Button().
													Class("pf-v5-c-button pf-m-primary").
													Type("button").
													Text("Refresh Kernels").
													Style("font-size", "16px").
													Style("margin-right", "16px").
													OnClick(func(ctx app.Context, e app.Event) {
														e.StopImmediatePropagation()
														app.Log("Refreshing kernels in kernel list.")

														go w.WorkloadDriver.KernelProvider().RefreshResources()
													}),
												app.Button().
													Class("pf-v5-c-button pf-m-secondary").
													Type("button").
													Text("Refresh Nodes").
													Style("font-size", "16px").
													Style("margin-right", "16px").
													OnClick(func(ctx app.Context, e app.Event) {
														e.StopImmediatePropagation()
														app.Log("Refreshing nodes in node list.")

														go w.WorkloadDriver.NodeProvider().RefreshResources()
													}),
												app.Button().
													Class("pf-v5-c-button pf-m-primary pf-m-danger").
													Type("button").
													Text("Terminate Selected Kernels").
													Style("font-size", "16px").
													OnClick(func(ctx app.Context, e app.Event) {
														e.StopImmediatePropagation()

														// kernelsToTerminate := make([]*gateway.DistributedJupyterKernel, 0, len(kernels))

														// for kernel_id, selected := range kl.selected {
														// 	if selected {
														// 		app.Logf("Kernel %s is selected. Will be terminating it.", kernel_id)
														// 		kernelsToTerminate = append(kernelsToTerminate, kernels[kernel_id])
														// 	}
														// }

														// app.Logf("Terminating %d kernels now.", len(kernelsToTerminate))
													}),
											),
									)),
						)),
			app.Div().Class("pf-v5-l-grid pf-m-gutter").Body(
				app.Div().Class("pf-v5-l-grid__item pf-m-gutter pf-m-6-col").Body(
					NewKernelList(w.WorkloadDriver, w, w.onMigrateButtonClicked),
				),
				app.Div().Class("pf-v5-l-grid__item pf-m-gutter pf-m-6-col").Body(
					NewNodeList(w.WorkloadDriver, w, false, func(kn *domain.KubernetesNode) { /* Do nothing */ }),
				),
			))
	} else {
		return app.Div()
	}
}

// The Render method is where the component appearance is defined.
func (w *MainWindow) Render() app.UI {
	app.Logf("Rendering MainWindow (%p). ConfigurationReceived: %v. WorkloadDriver: %v.", w, w.ConfigurationReceived, w.WorkloadDriver)

	// linkClass := "link heading fit unselectable"
	return app.Div().Body(
		app.Div().
			Class("pf-v5-c-page").
			Body(
				app.Main().
					Class("pf-v5-c-page__main").
					ID("driver-main").
					TabIndex(-1).
					Body(
						app.Section().Class("pf-v5-c-page__main-section").Body(
							app.Div().Class("pf-v5-c-page__main-body").Body(
								app.Div().Class("pf-v5-c-content").Body(
									app.If(w.WorkloadDriver == nil || !w.ConfigurationReceived, w.getPreConfigReceivedUI()),
									app.If(w.WorkloadDriver != nil && !w.WorkloadDriver.ConnectedToGateway(), w.getPreGatewayConnectionUI()),
									app.If(w.WorkloadDriver != nil && w.WorkloadDriver.ConnectedToGateway(), w.getUI()),
								))),
					),
				app.Section().
					Class("").
					Body(
						app.If(w.migarateModalOpen, &MigrationModal{
							ID:             "migrate-modal",
							WorkloadDriver: w.WorkloadDriver,
							ErrorHandler:   w,
							Icon:           "fas",
							Title:          "Migrate Kernel Replica",
							Class:          "pf-m-primary",
							Body:           "Migarte a kernel replica",
							ActionLabel:    "Submit",
							Replica:        w.replicaToMigrate,
							OnSubmit:       w.onMigrateSubmit,
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
			),
		app.If(w.Alerts.Len() > 0, &AlertList{
			Alerts: w.Alerts,
		}),
	)
}
