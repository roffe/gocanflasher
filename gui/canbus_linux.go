package gui

import (
	"context"
	"github.com/roffe/gocan"
)

func (m *mainWindow) initCAN(ctx context.Context) (*gocan.Client, error) {
	port := state.port
	return m.secondInit(ctx, port)
}
