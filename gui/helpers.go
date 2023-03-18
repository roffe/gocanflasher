package gui

import "github.com/roffe/gocanflasher/pkg/ecu"

func (m *mainWindow) setECU(t ecu.Type) {
	state.ecuType = t
	m.ecuList.SetSelected(t.String())
}
