package devices

import "github.com/LeFinal/masc-server/networking"

type DeviceManager struct {
	Hub *networking.Hub
}

func NewDeviceManager() *DeviceManager {
	return &DeviceManager{
		Hub: networking.NewHub(),
	}
}

func (dm *DeviceManager) Run() {
	dm.Hub.Run()
}
