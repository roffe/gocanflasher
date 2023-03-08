package gui

import (
	"context"
	"fmt"
	"time"

	"github.com/roffe/gocan"
	"github.com/roffe/gocan/adapter"
	"github.com/roffe/gocanflasher/pkg/ecu"
)

func (m *mainWindow) secondInit(ctx context.Context, port string) (*gocan.Client, error) {
	startTime := time.Now()
	m.output("Init adapter")
	defer func() {
		m.output(fmt.Sprintf("Done, took: %s", time.Since(startTime).Round(time.Millisecond).String()))
	}()
	dev, err := adapter.New(
		state.adapter,
		&gocan.AdapterConfig{
			Port:         port,
			PortBaudrate: state.portBaudrate,
			CANRate:      state.canRate,
			CANFilter:    ecu.CANFilters(state.ecuType),
			OutputFunc:   m.output,
			ErrorFunc: func(err error) {
				m.output(fmt.Sprintf("Error: %s", err.Error()))
			},
		})
	if err != nil {
		return nil, err
	}

	return gocan.New(ctx, dev)
}
