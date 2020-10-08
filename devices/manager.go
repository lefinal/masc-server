package devices

type DeviceManager struct {
	Devices map[*Device]bool
}

func NewDeviceManager() *DeviceManager {
	return &DeviceManager{
		Devices: make(map[*Device]bool),
	}
}

func (d *DeviceManager) AcceptNewDevice(device *Device) {
	panic("implement me")
}
