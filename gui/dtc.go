package gui

import (
	"context"
	"fmt"
	"time"

	"github.com/roffe/gocanflasher/pkg/ecu"
)

func (m *mainWindow) readDTC() {
	if !m.checkSelections() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		state.inprogress = true
		defer func() {
			state.inprogress = false
		}()

		m.disableButtons()
		defer m.enableButtons()

		c, err := m.initCAN(ctx)
		if err != nil {
			m.output(err.Error())
			return
		}
		defer c.Close()

		tr, err := ecu.New(c, &ecu.Config{
			Name:       state.ecuType,
			OnProgress: m.progress,
			OnMessage:  m.output,
			OnError:    m.error,
		})
		if err != nil {
			m.output(err.Error())
			return
		}

		dtcs, err := tr.ReadDTC(ctx)
		if err != nil {
			m.output(err.Error())
			return

		}

		if len(dtcs) == 0 {
			m.output("No DTC's")
		} else {
			m.output("Detected DTC's:")
		}

		for i, dtc := range dtcs {
			m.output(fmt.Sprintf("#%d %s", i, dtc.String()))
		}

	}()
}
