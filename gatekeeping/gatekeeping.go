package gatekeeping

import (
	"github.com/LeFinal/masc-server/config"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/networking"
)

var logger = logging.NewLogger("gatekeeping")

// Employer acts as a general interface for the gatekeeper.
type Employer interface {
	AcceptNewGatePort(port *GatePort)
}

// Gatekeeper is responsible for managing login and ensuring that a device sends data with correct meta.
type Gatekeeper struct {
	Hub                *networking.Hub
	customer           Employer
	deviceNetworkPorts []*GatePort
}

// NewGateKeeper creates a new GateKeeper using the network config.
func NewGateKeeper(config config.NetworkConfig, customer Employer) *Gatekeeper {
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
			logger.Info("Handling new client...")
			gk.handleNewClient(newClient)
		case closedClient := <-gk.Hub.ClosedClients:
			logger.Info("Handling closed client...")
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
