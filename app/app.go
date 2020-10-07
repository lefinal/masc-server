package app

import "github.com/LeFinal/masc-server/devices"

type App struct {
	deviceManager *devices.DeviceManager
}

func NewApp() *App {
	return &App{
		deviceManager: devices.NewDeviceManager(),
	}
}

func (a *App) boot() error {
	go a.deviceManager.Run()
	return nil
}
