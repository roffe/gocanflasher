package gui

import (
	"context"

	"github.com/roffe/gocan"
	"github.com/roffe/gocan/adapter/passthru"
)

func (m *mainWindow) initCAN(ctx context.Context) (*gocan.Client, error) {
	port := state.port
	if state.adapter == "J2534" {
		for _, p := range passthru.FindDLLs() {
			if p.Name == state.port {
				port = p.FunctionLibrary
				break
			}
		}
	}
	return m.secondInit(ctx, port)
}
