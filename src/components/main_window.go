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

var (
	// This will be updated once we receive the configuration file.
	// Specifically, it will be set to whatever the GatewayAddress configuration parameter is.
	DefaultGatewayAddress string = "127.0.0.1:9000"
)

type MigrateButtonClickedHandler func(app.Context, app.Event, *gateway.JupyterKernelReplica)

type MainWindow struct {
	app.Compo

	GatewayAddress        string                                 // Address of the Cluster Gateway, either set to the default value or populated by the associated input text box.
	Alerts                *orderedmap.OrderedMap[string, *Alert] // Current, non-dismissed alerts. These are displayed to the user.
	ConfigurationReceived bool                                   // Flag indicating whether we've received the configuration from the server.
	WorkloadDriver        domain.WorkloadDriver                  // The Workload Driver.

	configuration     *config.Configuration         // The system configuration sent to us by the backend server.
	errMsg            string                        // Current error message.
	err               error                         // Current operational error.
	updateAvailable   bool                          // Field that reports whether an app update is available. False by default.
	replicaToMigrate  *gateway.JupyterKernelReplica // The replica for which the user has clicked the 'Migrate' button.
	migarateModalOpen bool                          // Indicates whether the MigrateModal should be open. If it is true, then the migrate modal will be displayed.
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

	ctx.Dispatch(func(ctx app.Context) {
		w.onConfigReceived(&opts)
	})
}

// Called when we receive the configuration from the backend.
func (w *MainWindow) onConfigReceived(configuration *config.Configuration) {
	app.Logf(fmt.Sprintf("Received configuration:\n%s", configuration.String()))

	w.configuration = configuration
	DefaultGatewayAddress = configuration.GatewayAddress

	driver := driver.NewWorkloadDriver(w, configuration)
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

// Phase 1:
// - We're in "Phase 1" if either of the following are true (i.e., a OR b):
//   - (a) Workload Driver has not been created yet.
//   - (b) We've not yet received the configuration from the backend server.
func (w *MainWindow) checkPhaseOneUICondition() bool {
	return w.WorkloadDriver == nil || !w.ConfigurationReceived
}

// Return the UI that is to be displayed during Phase 1.
// See the MainWindow::checkPhaseOneUICondition function for details about what Phase 1 is.
func (w *MainWindow) getPhaseOneUI() app.UI {
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
}

// Phase 2:
// - We're in "Phase 2" if BOTH of the following are true (i.e., a AND b):
//   - (a) We've received the configuration from the backend server.
//   - (b) We've not yet connected to the Gateway. If we're spoofing the Gateway connection, then this part is satisfied immediately/automatically/inherently.
func (w *MainWindow) checkPhaseTwoUICondition() bool {
	// We need to have received the configuration. That's condition (a).
	if w.ConfigurationReceived {
		// For condition (b), there are two things we can check here.
		// If we're supposed to spoof the gateway connection, then we essentially skip Phase 2 and can move right to Phase 3.
		// If we're NOT supposed to spoof the gateway connection -- that is, we're actually supposed to connect, then we need to check if we've connected.
		// If we have not yet connected, then we are indeed in Phase 2  then we'll return true. If not, then we'll return false.
		if w.configuration.SpoofCluster {
			// We return false because we're essentially skipping phase 2.
			// We're spoofing the gateway connection, so we can just skip past the UI for specifying the gateway's address.
			return false
		}

		// If we're NOT spoofing the connection, then we're only in Phase 2 if we've NOT YET CONNECTED.
		// If we have already connected, then we're in Phase 3, not Phase 2.
		return !w.WorkloadDriver.ConnectedToGateway()
	}

	return false
}

// Return the UI that is to be displayed during Phase 2.
// See the MainWindow::checkPhaseTwoUICondition function for details about what Phase 2 is.
func (w *MainWindow) getPhaseTwoUI() app.UI {
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
								w.GatewayAddress = DefaultGatewayAddress
							}

							app.Logf("Connect clicked! Attempting to connect to Gateway (via gRPC) at %s now...", w.GatewayAddress)
							go w.connectButtonHandler()
						}),
				),
		)
}

// Phase 3:
// - We're in "Phase 3" if BOTH of the following are true (i.e., a AND b):
//   - (a) We've received the configuration from the backend server.
//   - (b) We've not yet connected to the Gateway. If we're spoofing the Gateway connection, then this part is satisfied immediately/automatically/inherently.
func (w *MainWindow) checkPhaseThreeUICondition() bool {
	// We need to have received the configuration. That's condition (a).
	if w.ConfigurationReceived {
		// For condition (b), there are two things we can check here.
		// If we're supposed to spoof the gateway connection, then we can immediately conclude that we're in Phase 3.
		// If we're NOT supposed to spoof the gateway connection -- that is, we're actually supposed to connect, then we need to check if we've connected.
		// In order for us to be in Phase 3, we need to be connected to the cluster gateway.
		if w.configuration.SpoofCluster {
			// If we're supposed to spoof the gateway connection, then we can immediately conclude that we're in Phase 3.
			return true
		}

		// If we're NOT spoofing the connection, then we're only in Phase 2 if we ARE already connected.
		return w.WorkloadDriver.ConnectedToGateway()
	}

	return false
}

// Return the UI that is to be displayed during Phase 3.
// See the MainWindow::checkPhaseTwoUICondition function for details about what Phase 3 is.
func (w *MainWindow) getPhaseThreeUI() app.UI {
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
}

// This function returns the correct UI to render based on several conditions.
// I previously just used app.If() for this; however, app.If() always evaluates the elements to be returned, even if the condition is false.
// This is bad for performance, and is also somewhat unintuitive and error-prone.
// In particular, several of our conditions as well as parts of the UI itself depend upon whether the MainWindow::WorkloadDriver field is nil.
// Thus, we are liable to panic due to a null pointer exception if we use app.If().
func (w *MainWindow) getUI() app.UI {
	if w.checkPhaseOneUICondition() {
		return w.getPhaseOneUI()
	} else if w.checkPhaseTwoUICondition() {
		return w.getPhaseTwoUI()
	} else if w.checkPhaseThreeUICondition() {
		return w.getPhaseThreeUI()
	} else {
		app.Log("Unable to determine which UI to display to the user.")

		w.err = fmt.Errorf("unable to determine which UI to display to the user")
		w.errMsg = "The webpage is unable to determine which UI to display to the user. There is a bug in the UI-condition code within the MainWindow."

		// I could just panic here instead, but the ErrorModal will be more immediately obvious, in theory...

		return app.Div()
	}
}

// The Render method is where the component appearance is defined.
func (w *MainWindow) Render() app.UI {
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
									w.getUI(),
								),
							),
						),
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
