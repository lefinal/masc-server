package main

import (
	"github.com/LeFinal/masc-server/app"
	"github.com/LeFinal/masc-server/config"
)

func main() {
	masc := app.NewApp(config.MascConfig{
		// TODO
	})
	masc.Boot()
}
