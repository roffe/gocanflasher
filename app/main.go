package main

import (
	"context"
	_ "embed"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/roffe/gocanflasher/gui"

	// Import ecu packages
	_ "github.com/roffe/gocanflasher/pkg/ecu/t5"
	_ "github.com/roffe/gocanflasher/pkg/ecu/t5legion"
	_ "github.com/roffe/gocanflasher/pkg/ecu/t7"
	_ "github.com/roffe/gocanflasher/pkg/ecu/t8"
	_ "github.com/roffe/gocanflasher/pkg/ecu/t8mcp"
)

//go:embed Icon.png
var appIconBytes []byte

var appIcon = fyne.NewStaticResource("ecu.png", appIconBytes)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func main() {
	a := app.NewWithID("GoCANFlasher")
	a.Settings().SetTheme(&MyTheme{})
	a.SetIcon(appIcon)
	gui.ShowAndRun(context.TODO(), a)
}
