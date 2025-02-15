package gui

import (
	"context"
	"strconv"
	"time"

	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	sdialog "github.com/sqweek/dialog"
)

type appState struct {
	ecuType string
	//canRate      float64
	adapter      string
	port         string
	portBaudrate int
	portList     []string
	inprogress   bool
}

var (
	listData = binding.NewStringList()
	state    = new(appState)
)

func ShowAndRun(ctx context.Context, a fyne.App) {
	w := a.NewWindow("GoCANFlasher")

	mw := newMainWindow(a, w)

	go func() {
		<-ctx.Done()
		time.Sleep(500 * time.Millisecond)
		w.Close()
	}()

	mw.loadPreferences()

	go func() {
		time.Sleep(10 * time.Millisecond)
		mw.output("Done detecting ports")
	}()

	w.ShowAndRun()
}

func (m *mainWindow) closeHandler() {
	if state.inprogress {
		if sdialog.Message("Are you sure, operation still in progress").Title("In progress").YesNo() {
			m.window.Close()
		}
		return
	}
	m.wizzardWindow.window.Close()
	m.window.Close()
}

func (m *mainWindow) loadPreferences() {
	//state.canRate = m.app.Preferences().FloatWithFallback("canrate", 500)
	m.ecuList.SetSelectedIndex(m.app.Preferences().IntWithFallback("ecu", 0))
	m.adapterList.SetSelected(m.app.Preferences().StringWithFallback("adapter", "Canusb"))
	state.port = m.app.Preferences().String("port")
	m.portList.PlaceHolder = state.port
	m.portList.Refresh()
	m.speedList.SetSelected(m.app.Preferences().StringWithFallback("portSpeed", "115200"))
}

func speeds() []string {
	var out []string
	l := []int{9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600, 1000000, 2000000, 3000000}
	for _, ll := range l {
		out = append(out, strconv.Itoa(ll))
	}
	return out
}
