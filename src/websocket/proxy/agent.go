package proxy

import (
	"github.com/hashicorp/consul/api"
	"log"
)

func startProxy() {
	// Get a new client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	// Discover websocket servers
	services, _, err := client.Catalog().Service("websocket-server", "", nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, service := range services {
		log.Println("Found server:", service.ServiceID, "at", service.ServiceAddress, ":", service.ServicePort)
	}

	// Your gateway logic here
}
