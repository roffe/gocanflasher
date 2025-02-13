package t5legion

import (
	"context"
	"fmt"
	"time"

	"github.com/roffe/gocan"
)

func (t *Client) EraseECU(ctx context.Context) error {
	startTime := time.Now()
	if !t.bootloaded {
		if err := t.UploadBootLoader(ctx); err != nil {
			return err
		}
	}
	t.cfg.OnProgress(-float64(100))
	t.cfg.OnMessage("Erasing FLASH...")

	cmd := []byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	frame := gocan.NewFrame(0x005, cmd, gocan.ResponseRequired)
	resp, err := t.c.SendAndWait(ctx, frame, 20*time.Second, 0xC)
	if err != nil {
		return err
	}
	data := resp.Data()
	if data[0] == 0xC0 && data[1] == 0x00 {
		t.cfg.OnMessage(fmt.Sprintf("FLASH erased, took: %s\n", time.Since(startTime).Round(time.Millisecond).String()))
		t.cfg.OnProgress(float64(100))
		return nil
	}

	return fmt.Errorf("erase FAILED: %X", data)
}
