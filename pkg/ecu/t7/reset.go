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
	d := f.Data()
	if d[3] == 0x7F {
		return fmt.Errorf("failed to reset ECU: %s", TranslateErrorCode(d[5]))
	}
	if d[3] != 0x51 || d[4] != 0x81 {
		return fmt.Errorf("abnormal ecu reset response: %X", d[3:])
	}
	return nil
}
