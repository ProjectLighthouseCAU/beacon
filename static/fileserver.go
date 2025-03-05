package static

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ProjectLighthouseCAU/beacon/config"
)

func StartFileserver() {
	// serve static testing site (only works with websocket endpoint enabled)
	log.Printf("Serving static files: http://%s:%d%s\n", config.WebinterfaceHost, config.WebinterfacePort, config.WebinterfaceRoute)
	mux := http.NewServeMux()
	mux.Handle(config.WebinterfaceRoute, http.FileServer(http.Dir("./static")))
	serv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.WebinterfaceHost, config.WebinterfacePort),
		Handler: mux,
	}
	go func() {
		log.Fatal(serv.ListenAndServe())
	}()
}
