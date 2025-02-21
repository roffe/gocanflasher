package t5

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/roffe/gocan"
)

func (t *Client) ResetECU(ctx context.Context) error {
	//if !t.bootloaded {
	//	if err := t.UploadBootLoader(ctx); err != nil {
	//		return err
	//	}
	//}
	//log.Println("Resetting ECU")
	frame := gocan.NewFrame(0x5, []byte{0xC2, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, gocan.ResponseRequired)
	resp, err := t.c.SendAndWait(ctx, frame, 150*time.Millisecond, 0xC)
	if err != nil {
		return fmt.Errorf("failed to reset ECU: %v", err)
	}
	if resp.Data[0] != 0xC2 || resp.Data[1] != 0x00 || resp.Data[2] != 0x08 {
		return errors.New("invalid response to reset ECU")
	}
	t.cfg.OnMessage("ECU has been reset")
	return nil
}
