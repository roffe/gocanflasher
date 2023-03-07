package gui

func (m *mainWindow) initCAN(ctx context.Context) (*gocan.Client, error) {
	return secondInit(ctx, state.port)
}
