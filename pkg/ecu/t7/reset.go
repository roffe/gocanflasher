package t7

import (
	"context"
	"fmt"

	"github.com/roffe/gocan"
)

// Noop command to satisfy interface
func (t *Client) ResetECU2(ctx context.Context) error {
	return nil
}

func (t *Client) ResetECU(ctx context.Context) error {
	frame := gocan.NewFrame(0x240, []byte{0x40, 0xA1, 0x02, 0x11, 0x01}, gocan.ResponseRequired)
	f, err := t.c.SendAndWait(ctx, frame, t.defaultTimeout, 0x258)
	if err != nil {
		return err
	}
	if f.Data[3] == 0x7F {
		return fmt.Errorf("failed to reset ECU: %s", TranslateErrorCode(f.Data[5]))
	}
	if f.Data[3] != 0x51 || f.Data[4] != 0x81 {
		return fmt.Errorf("abnormal ecu reset response: %X", f.Data[3:])
	}
	return nil
}
