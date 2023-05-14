package t5legion

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
	frame := gocan.NewFrame(0x5, []byte{0x01, 0x20}, gocan.ResponseRequired)
	resp, err := t.c.SendAndPoll(ctx, frame, 150*time.Millisecond, 0xC)
	if err != nil {
		return fmt.Errorf("failed to reset ECU: %v", err)
	}
	data := resp.Data()
	if data[0] != 0x01 || data[1] != 0x50 && data[1] != 0x60 {
		return errors.New("invalid response to reset ECU")
	}
	t.cfg.OnMessage("ECU has been reset")
	return nil
}
