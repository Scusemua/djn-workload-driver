package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

func getIconForKernelStatusLabel(status string) string {
	switch status {
	case "unknown":
		return "fas fa-question"
	case "starting":
		return "fas fa-spinner fa-pulse fa-spin"
	case "idle":
		return "fas fa-pause"
	case "busy":
		return "fas fa-hourglass-half"
	case "terminating":
		return "fas fa-spinner fa-pulse fa-spin"
	case "restarting":
		return "fas fa-spinner fa-pulse fa-spin"
	case "autorestarting":
		return "fas fa-spinner fa-pulse fa-spin"
	case "dead":
		return "fas fa-skull"
	default:
		app.Logf("[WARNING] Unknown kernel status received: \"%s\"\n", status)
		return ""
	}
}

// Displays the aggregate status of a kernel in a KernelList.
type KernelStatusLabel struct {
	app.Compo

	status   string
	fontSize int
}

func NewKernelStatusLabel(status string, fontSize int) *KernelStatusLabel {
	return &KernelStatusLabel{
		status:   status,
		fontSize: fontSize,
	}
}

func (ks *KernelStatusLabel) Render() app.UI {
	return app.Div().
		Class("pf-l-flex pf-m-space-items-xs").
		Body(
			app.Div().
				Class("pf-l-flex pf-m-space-items-xs").
				Body(
					app.Div().
						Class("pf-l-flex__item").
						Body(
							app.I().
								Class(getIconForKernelStatusLabel(ks.status)).
								Style("font-size", fmt.Sprintf("%spx", ks.fontSize)).
								Aria("hidden", true),
						),
					app.Div().
						Class("pf-l-flex__item").
						Body(
							app.Span().
								Text(ks.status).
								Style("font-size", fmt.Sprintf("%spx", ks.fontSize)),
						),
				),
		)
}
