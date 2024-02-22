package driver

import (
	"errors"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

// ErrorModal is a modal to display an error
type ErrorModal struct {
	app.Compo

	ID          string // HTML ID of the modal; must be unique across the page
	Icon        string // Class of the icon to use to the left of the title; may be empty
	Title       string // Title of the modal
	Class       string // Class to be applied to the modal's outmost component
	Body        string // Body text of the modal
	Error       error  // The error display (must not be nil)
	ActionLabel string // Text to display on the modal's primary action

	OnClose  func() // Handler to call when closing/cancelling the modal
	OnAction func() // Handler to call when triggering the modal's primary action
}

type ErrorModalStory struct {
	app.Compo
	modalOpen    bool
	errorMessage string
	alertBody    string
	onClose      func()
}

func NewErrorModalStory(errorMessage string, err error, modalOpen bool, onClose func()) *ErrorModalStory {
	if err == nil {
		return &ErrorModalStory{
			modalOpen:    modalOpen,
			errorMessage: errorMessage,
			alertBody:    "An unknown error has occurred.",
			onClose:      onClose,
		}
	} else {
		return &ErrorModalStory{
			modalOpen:    modalOpen,
			errorMessage: errorMessage,
			alertBody:    err.Error(),
			onClose:      onClose,
		}
	}
}

func (c *ErrorModalStory) Render() app.UI {
	return app.Div().Body(
		app.Button().
			Class("pf-c-button pf-m-primary").
			Type("button").
			Text("Open error modal").
			OnClick(func(ctx app.Context, e app.Event) {
				c.modalOpen = !c.modalOpen
			}),
		app.If(
			c.modalOpen,
			app.UI(
				&ErrorModal{
					ID:          "error-modal-story",
					Icon:        "fas fa-times",
					Title:       "Error",
					Class:       "pf-m-danger",
					Body:        c.alertBody,
					Error:       errors.New(c.errorMessage),
					ActionLabel: "Close",

					OnClose: func() {
						c.modalOpen = false
						c.onClose()
						c.Update()
					},
					OnAction: func() {
						c.modalOpen = false
						c.Update()
					},
				},
			)),
	)
}
