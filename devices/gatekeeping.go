package devices

import (
	"github.com/LeFinal/masc-server/config"
	"github.com/LeFinal/masc-server/networking"
)

// GatekeepingCustomer acts as a general interface for the gatekeeper and the device manager.
type GatekeepingCustomer interface {
	AcceptNewDevice(*Device)
}

type Gatekeeper struct {
	Hub      *networking.Hub
	Customer GatekeepingCustomer
}

func NewGateKeeper(config config.NetworkConfig) *Gatekeeper {
	return &Gatekeeper{
		Hub: networking.NewHub(config.Address),
	}
}

func (gk *Gatekeeper) Start() {
	// Start hub
	go gk.Hub.Run()
}

func (gk *Gatekeeper) run() {

}
