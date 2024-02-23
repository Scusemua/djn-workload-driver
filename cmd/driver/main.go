package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/scusemua/djn-workload-driver/m/v2/src/components"
	"github.com/scusemua/djn-workload-driver/m/v2/src/driver"
)

func main() {
	app.RouteFunc("/", func() app.Composer {
		mainWindow := &components.MainWindow{}
		driver := driver.NewWorkloadDriver(mainWindow, true, time.Second*5)
		mainWindow.SetWorkloadDriver(driver)
		driver.Start()
		return mainWindow
	})

	// Once the routes set up, the next thing to do is to either launch the app
	// or the server that serves the app.
	//
	// When executed on the client-side, the RunWhenOnBrowser() function
	// launches the app,  starting a loop that listens for app events and
	// executes client instructions. Since it is a blocking call, the code below
	// it will never be executed.
	//
	// When executed on the server-side, RunWhenOnBrowser() does nothing, which
	// lets room for server implementation without the need for precompiling
	// instructions.
	app.RunWhenOnBrowser()

	// Finally, launching the server that serves the app is done by using the Go
	// standard HTTP package.
	//
	// The Handler is an HTTP handler that serves the client and all its
	// required resources to make it work into a web browser. Here it is
	// configured to handle requests with a path that starts with "/".
	http.Handle("/", &app.Handler{
		Name:        "WorkloadDriver",
		Description: "Workload Driver for the Distributed Jupyter Notebook platform.",
		Styles: []string{
			"/web/main.css",
			"/web/css/docs.css",
		},
		Icon: app.Icon{
			SVG: "/web/icon.svg",
		},
	})

	fmt.Printf("WorkloadDriver HTTP server is starting now.")

	// TODO(Ben): Make this port configurable.
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}
