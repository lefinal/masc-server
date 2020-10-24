package gatekeeping

import (
	"github.com/LeFinal/masc-server/config"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/networking"
)

var logger = logging.NewLogger("gatekeeping")

// Employer acts as a general interface for the gatekeeper.
type Employer interface {
	MessageAcceptor
	AcceptNewGatePort(port *GatePort)
	HandleClosedGatePort(port *GatePort)
}

// MessageAcceptor is able to accept any new message.
type MessageAcceptor interface {
	AcceptNewMessage(container messages.MessageContainer)
}

// Gatekeeper is responsible for managing login and ensuring that a device sends data with correct meta.
type Gatekeeper struct {
	Hub                *networking.Hub
	employer           Employer
	deviceNetworkPorts map[*GatePort]bool
}

// NewGateKeeper creates a new GateKeeper using the network config.
func NewGateKeeper(config config.NetworkConfig, employer Employer) *Gatekeeper {
	return &Gatekeeper{
		Hub:                networking.NewHub(config),
		employer:           employer,
		deviceNetworkPorts: make(map[*GatePort]bool),
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

// handleNewClient handles a new client, creates a network port, starts it and tells the employer.
func (gk *Gatekeeper) handleNewClient(c networking.Client) {
	gatePort := newGatePort(c, gk.employer)
	gk.deviceNetworkPorts[gatePort] = true
	gk.employer.AcceptNewGatePort(gatePort)
	go gatePort.run()
}

// handleClosedClient handles a closed client, removes it's port from the device network ports and tells the employer.
func (gk *Gatekeeper) handleClosedClient(c networking.Client) {
	// Find the port for the given client.
	var port *GatePort
	for currentPort, _ := range gk.deviceNetworkPorts {
		if currentPort.client == c {
			port = currentPort
		}
	}
	if port == nil {
		logger.MascError(errors.NewMascError("find port for closed client", errors.UnknownClientError))
		return
	}
	// Remove and stop the port.
	delete(gk.deviceNetworkPorts, port)
	port.stop()
	// Notify the employer.
	gk.employer.HandleClosedGatePort(port)
}
