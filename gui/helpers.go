package gui

func (m *mainWindow) setECU(t string) {
	state.ecuType = t
	m.ecuList.SetSelected(t)
}
