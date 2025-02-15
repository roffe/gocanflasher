package t5

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/avast/retry-go"
)

func (t *Client) DumpECU(ctx context.Context) ([]byte, error) {
	if !t.bootloaded {
		if err := t.UploadBootLoader(ctx); err != nil {
			return nil, err
		}
	}

	ecutype, err := t.DetermineECU(ctx)
	if err != nil {
		return nil, err
	}

	start := getstartAddress(ecutype)
	length := 0x80000 - start

	t.cfg.OnProgress(-float64(length))
	t.cfg.OnMessage("Dumping ECU")

	buffer := make([]byte, length)
	startTime := time.Now()
	progress := 0

	address := start + 5
	for i := 0; i < int(length/6); i++ {
		err := retry.Do(
			func() error {
				b, err := t.ReadMemoryByAddress(ctx, address)
				if err != nil {
					return err
				}
				copy(buffer[i*6:], b)
				progress += len(b)
				return nil
			},
			retry.OnRetry(func(n uint, err error) {
				t.cfg.OnError(fmt.Errorf("retrying to read memory by address: %w", err))
			}),
			retry.Attempts(3),
			retry.LastErrorOnly(true),
		)
		if err != nil {
			return nil, err
		}
		address += 6
		t.cfg.OnProgress(float64(progress))
	}

	// Get the leftover bytes
	if (length % 6) > 0 {
		err := retry.Do(
			func() error {
				b, err := t.ReadMemoryByAddress(ctx, start+length-1)
				if err != nil {
					return err
				}
				for j := (6 - (length % 6)); j < 6; j++ {
					buffer[length-6+j] = b[j]
					progress++
				}
				return nil
			},
			retry.OnRetry(func(n uint, err error) {
				t.cfg.OnError(fmt.Errorf("t5 retrying to read memory by address: %w", err))
			}),
			retry.Attempts(3),
			retry.LastErrorOnly(true),
		)
		if err != nil {
			return nil, err
		}
		t.cfg.OnProgress(float64(progress))
	}

	t.cfg.OnProgress(float64(length))
	t.cfg.OnMessage(fmt.Sprintf("Done, took: %s", time.Since(startTime).Round(time.Millisecond).String()))

	checksum, err := t.GetECUChecksum(ctx)
	if err != nil {
		return nil, err
	}

	calculated, err := t.CalculateBinChecksum(buffer)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(checksum, calculated) {
		t.cfg.OnError(errors.New("Dumped bin and calculated checksum from ECU does not match")) //lint:ignore ST1005 ignore this shit
		t.cfg.OnError(fmt.Errorf("ECU reported checksum: %X, calculated: %X", checksum, calculated))
	}

	return buffer, nil
}

func getstartAddress(ecutype ECUType) uint32 {
	switch ecutype {
	case T52ECU, T55AST52:
		return 0x60000
	default:
		return 0x40000
	}
}
