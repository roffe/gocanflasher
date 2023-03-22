package t8

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"time"

	"github.com/roffe/gocanflasher/pkg/legion"
)

func (t *Client) DumpECU(ctx context.Context) ([]byte, error) {
	if err := t.legion.Bootstrap(ctx); err != nil {
		return nil, err
	}

	t.cfg.OnMessage("Dumping ECU")
	start := time.Now()

	bin, err := t.legion.ReadFlash(ctx, legion.EcuByte_T8, 0x100000, false)
	if err != nil {
		return nil, err
	}

	t.cfg.OnMessage("Verifying md5..")

	ecuMD5bytes, err := t.legion.IDemand(ctx, legion.GetTrionic8MD5, 0x00)
	if err != nil {
		return nil, err
	}
	calculatedMD5 := md5.Sum(bin)

	t.cfg.OnMessage(fmt.Sprintf("Remote MD5 : %X", ecuMD5bytes))
	t.cfg.OnMessage(fmt.Sprintf("Local MD5  : %X", calculatedMD5))

	if !bytes.Equal(ecuMD5bytes, calculatedMD5[:]) {
		return nil, errors.New("md5 Verification failed")
	}

	t.cfg.OnMessage("Done, took: " + time.Since(start).String())

	return bin, nil
}
