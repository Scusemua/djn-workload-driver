package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	corev1 "k8s.io/api/core/v1"
)

func getIconForPodStatusLabel(status string, fontSize int) app.UI {
	switch status {
	case string(corev1.PodPending):
		return app.I().
			Class("fas fa-clock").
			Style("font-size", fmt.Sprintf("%dpx", fontSize)).
			Aria("hidden", true)
	case string(corev1.PodRunning):
		return app.I().
			Class("fas fa-check pf-v5-u-success-color-100").
			Style("font-size", fmt.Sprintf("%dpx", fontSize)).
			Aria("hidden", true)
	case string(corev1.PodSucceeded):
		return app.I().
			Class("fas fa-check-circle pf-v5-u-success-color-100").
			Style("font-size", fmt.Sprintf("%dpx", fontSize)).
			Aria("hidden", true)
	case string(corev1.PodFailed):
		return app.I().
			Class("fas fa-times-circle pf-v5-u-danger-color-100").
			Style("font-size", fmt.Sprintf("%dpx", fontSize)).
			Aria("hidden", true)
	case string(corev1.PodUnknown):
		return app.I().
			Class("fas fa-exclamation-triangle pf-v5-u-warning-color-100").
			Style("font-size", fmt.Sprintf("%dpx", fontSize)).
			Aria("hidden", true)
	default:
		app.Logf("[WARNING] Unknown kernel status received: \"%s\"\n", status)
		return app.I().
			Class("fas fa-question").
			Style("font-size", fmt.Sprintf("%dpx", fontSize)).
			Aria("hidden", true)
	}
}

// Displays the aggregate status of a kernel in a KernelList.
type PodStatusLabel struct {
	app.Compo

	status   string
	fontSize int
}

func NewPodStatusLabel(status string, fontSize int) *PodStatusLabel {
	return &PodStatusLabel{
		status:   status,
		fontSize: fontSize,
	}
}

func (ks *PodStatusLabel) Render() app.UI {
	return app.Div().
		Class("pf-v5-l-flex pf-m-space-items-xs").
		Body(
			app.Div().
				Class("pf-v5-l-flex pf-m-space-items-xs").
				Body(
					app.Div().
						Class("pf-v5-l-flex__item").
						Body(
							getIconForPodStatusLabel(ks.status, ks.fontSize),
						),
					app.Div().
						Class("pf-v5-l-flex__item").
						Body(
							app.Span().
								Text(ks.status).
								Style("font-size", fmt.Sprintf("%dpx", ks.fontSize)),
						),
				),
		)
}
