package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

type ResourceLabel struct {
	app.Compo

	// If true, use the Class field to display the content.
	// If false, use the Content field to display the content.
	UseClass bool

	ResourceName string
	Content      string
	Class        string
	FontSize     int
	Allocated    float64
	Capacity     float64
}

func getPercentage(allocated float64, capacity float64) float64 {
	if capacity == 0 {
		return 0.0
	}

	return allocated / capacity
}

func (rl *ResourceLabel) Render() app.UI {
	if rl.UseClass {
		return app.Div().
			Class("pf-v5-l-flex pf-m-space-items-xs").
			Body(
				app.Div().
					Class("pf-v5-l-flex__item").
					Body(
						app.I().
							Class(rl.Class).
							Style("font-size", fmt.Sprintf("%dpx", rl.FontSize)).
							Aria("hidden", true),
					),
				app.Div().
					Class("pf-v5-l-flex__item").
					Body(
						app.Span().
							Text(fmt.Sprintf("%.2f / %.2f (%.2f%%)", rl.Allocated, rl.Capacity, getPercentage(rl.Allocated, rl.Capacity))).
							Style("font-size", fmt.Sprintf("%dpx", rl.FontSize)),
					),
			)
	} else {
		return app.Div().
			Class("pf-v5-l-flex pf-m-space-items-xs").
			Body(
				app.Div().
					Class("pf-v5-l-flex__item").
					Body(
						app.I().
							Style("content", fmt.Sprintf("url(\"%s\")", rl.Content)).
							Style("font-size", fmt.Sprintf("%dpx", rl.FontSize)).
							Aria("hidden", true),
					),
				app.Div().
					Class("pf-v5-l-flex__item").
					Body(
						app.Span().
							Text(fmt.Sprintf("%.2f / %.2f (%.2f%%)", rl.Allocated, rl.Capacity, getPercentage(rl.Allocated, rl.Capacity))).
							Style("font-size", fmt.Sprintf("%dpx", rl.FontSize)),
					),
			)
	}
}
