package gatekeeping

import (
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/ws"
)

// TODO
type NetGatekeeper struct {
}

func (gk *NetGatekeeper) WakeUpAndProtect(protected Protected) error {
	panic("implement me")
}

func (gk *NetGatekeeper) Retire() error {
	panic("implement me")
}

func (gk *NetGatekeeper) GetDevices() []*Device {
	panic("implement me")
}

func (gk *NetGatekeeper) AcceptDevice(deviceID messages.DeviceID, name string) error {
	panic("implement me")
}

func (gk *NetGatekeeper) AcceptClient(client *ws.Client) {
	panic("implement me")
}

func (gk *NetGatekeeper) SayGoodbyeToClient(client *ws.Client) {
	panic("implement me")
}
