package devices

import (
	"github.com/LeFinal/masc-server/config"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/networking"
)

// GatekeepingCustomer acts as a general interface for the gatekeeper and the device manager.
type GatekeepingCustomer interface {
	AcceptNewDevice(*Device)
}

// Gatekeeper is responsible for managing login and ensuring that a device sends data with correct meta.
type Gatekeeper struct {
	Hub      *networking.Hub
	customer GatekeepingCustomer
	devices  []*Device
}

// NewGateKeeper creates a new GateKeeper using the network config.
func NewGateKeeper(config config.NetworkConfig, customer GatekeepingCustomer) *Gatekeeper {
	return &Gatekeeper{
		Hub:      networking.NewHub(config.Address),
		customer: customer,
	}
}

// Start starts the Gatekeeper with all its network components.
func (gk *Gatekeeper) Start() {
	// Start hub
	go gk.Hub.Run()
	go gk.run()
}

func (gk *Gatekeeper) run() {
	for {
		select {
		case newClient := <-gk.Hub.NewClients:
			logging.Info("Handling new client...")
			gk.handleNewClient(newClient)
		case closedClient := <-gk.Hub.ClosedClients:
			logging.Info("Handling closed client...")
			gk.handleClosedClient(closedClient)
		}
	}
}

func (gk *Gatekeeper) handleNewClient(c *networking.Client) {
	// TODO create a new go routine for every device which does the checking and parsing stuff and then passes to manager
}

func (gk *Gatekeeper) handleClosedClient(c *networking.Client) {
	// TODO
}
