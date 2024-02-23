package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

// Displays the number of replicas a kernel has in a KernelList.
type KernelReplicasLabel struct {
	app.Compo

	numReplicas int32
	fontSize    int
}

func NewKernelReplicasLabel(numReplicas int32, fontSize int) *KernelReplicasLabel {
	return &KernelReplicasLabel{
		numReplicas: numReplicas,
		fontSize:    fontSize,
	}
}

func (kl *KernelReplicasLabel) Render() app.UI {
	return app.Div().
		Class("pf-l-flex pf-m-space-items-xs").
		Body(
			app.Div().
				Class("pf-l-flex__item").
				Body(
					app.I().
						Class("fas fa-cube").
						Style("font-size", fmt.Sprintf("%spx", kl.fontSize)).
						Aria("hidden", true),
				),
			app.Div().
				Class("pf-l-flex__item").
				Body(
					app.Span().
						Text(kl.numReplicas).
						Style("font-size", fmt.Sprintf("%spx", kl.fontSize)),
				),
		)
}
