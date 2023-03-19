package gui

import (
	"context"
	"fmt"
	"time"

	"github.com/roffe/gocanflasher/pkg/ecu"
)

func (m *mainWindow) ecuInfo() {
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
			Type:       state.ecuType,
			OnProgress: m.callback,
			OnMessage: func(msg string) {
				m.callback(msg)
			},
			OnError: func(err error) {
				m.callback("Error: " + err.Error())
			},
		})
		if err != nil {
			m.output(err.Error())
			return
		}

		val, err := tr.Info(ctx)
		if err != nil {
			m.output(err.Error())
		}

		for _, v := range val {
			m.output(v.String())
		}

		if err := tr.ResetECU(ctx); err != nil {
			m.output(err.Error())
			return
		}
	}()
}

func (m *mainWindow) output(s string) {
	var text string
	if s != "" {
		text = fmt.Sprintf("%s - %s\n", time.Now().Format("15:04:05.000"), s)
	}
	//logData = append(logData, text)
	listData.Append(text)
	m.log.Refresh()
	m.log.ScrollToBottom()
}
