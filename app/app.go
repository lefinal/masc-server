package app

import (
	"github.com/LeFinal/masc-server/config"
	"github.com/LeFinal/masc-server/devices"
)

type App struct {
	deviceManager *devices.DeviceManager
}

func NewApp(config config.MascConfig) *App {
	return &App{
		deviceManager: devices.NewDeviceManager(config.Network),
	}
}

func (a *App) Boot() error {
	a.deviceManager.Start()
	return nil
}
