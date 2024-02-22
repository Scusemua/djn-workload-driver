package components

import (
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/domain"
)

type MainWindow struct {
	app.Compo

	workloadDriver domain.WorkloadDriver // The Workload Driver.
	errMsg         string                // Current error message.
	err            error                 // Current operational error.
}

func (w *MainWindow) SetWorkloadDriver(driver domain.WorkloadDriver) {
	w.workloadDriver = driver
}

func (w *MainWindow) recover() {
	w.err = nil
	w.errMsg = ""
}

func (w *MainWindow) HandleError(err error, errMsg string) {
	w.err = err
	w.errMsg = errMsg
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
					app.If(!w.workloadDriver.ConnectedToGateway(),
						NewGatewayConnectionWindow(w.workloadDriver, w)).
						Else(NewKernelList(w.workloadDriver))),
			app.Section().
				Class("").
				Body(
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
