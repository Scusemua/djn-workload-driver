package components

import "github.com/maxence-charriere/go-app/v9/pkg/app"

// Displays the number of replicas a kernel has in a KernelList.
type KernelReplicas struct {
	app.Compo

	numReplicas int32
}

func NewKernelReplicas(numReplicas int32) *KernelReplicas {
	return &KernelReplicas{
		numReplicas: numReplicas,
	}
}

func (ks *KernelReplicas) Render() app.UI {
	return app.Div().
		Class("pf-l-flex pf-m-space-items-xs").
		Body(
			app.Div().
				Class("pf-l-flex__item").
				Body(
					app.I().
						Class("fas fa-cube").
						Style("font-size", "24px").
						Aria("hidden", true),
				),
			app.Div().
				Class("pf-l-flex__item").
				Body(
					app.Span().
						Text(ks.numReplicas).
						Style("font-size", "24px"),
				),
		)
}
