package gui

import (
	"context"
	"fmt"
	"time"

	"github.com/roffe/gocan"
	"github.com/roffe/gocan/adapter"
	"github.com/roffe/gocan/adapter/passthru"
	"github.com/roffe/gocanflasher/pkg/ecu"
)

func (m *mainWindow) initCAN(ctx context.Context) (*gocan.Client, error) {
	startTime := time.Now()
	m.output("Init adapter")

	port := state.port

	if state.adapter == "J2534" {
		for _, p := range passthru.FindDLLs() {
			if p.Name == state.port {
				port = p.FunctionLibrary
				break
			}
		}
	}

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

	client, err := gocan.New(ctx, dev)
	if err != nil {
		return nil, err
	}

	m.output(fmt.Sprintf("Done, took: %s", time.Since(startTime).Round(time.Millisecond).String()))

	return client, nil
}
