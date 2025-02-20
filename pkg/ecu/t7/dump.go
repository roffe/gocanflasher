package t7

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/roffe/gocan"
)

func (t *Client) DumpECU(ctx context.Context) ([]byte, error) {
	ok, err := t.KnockKnock(ctx)
	if err != nil || !ok {
		return nil, fmt.Errorf("failed to authenticate: %v", err)
	}
	bin, err := t.readECU(ctx, 0, 0x80000)
	if err != nil {
		return nil, err
	}
	defer t.StopSession(ctx)
	return bin, nil
}

func (t *Client) readECU(ctx context.Context, addr, length int) ([]byte, error) {
	//addr := 0
	//length := 0x80000
	t.cfg.OnProgress(-float64(length))
	t.cfg.OnMessage("Dumping ECU")

	start := time.Now()
	var readPos int
	out := bytes.NewBuffer([]byte{})

	// if err := t.c.Adapter().SetFilter([]uint32{0x258}); err != nil {
	// 	t.cfg.OnError(err)
	// }

	for readPos < length {
		t.cfg.OnProgress(float64(out.Len()))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			readLength := min(0xF5, length-readPos)
			err := retry.Do(func() error {
				b, err := t.readMemoryByAddress(ctx, readPos, readLength)
				if err != nil {
					return err
				}
				out.Write(b)
				return nil
			},
				retry.Context(ctx),
				retry.Attempts(3),
				retry.OnRetry(func(n uint, err error) {
					t.cfg.OnMessage(fmt.Sprintf("Failed to read memory by address, pos: 0x%X, length: 0x%X, retrying: %v", readPos, readLength, err))
				}),
				retry.LastErrorOnly(true),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to read memory by address, pos: 0x%X, length: 0x%X", readPos, readLength)
			}
			readPos += readLength
		}
	}

	if err := t.endDownloadMode(ctx); err != nil {
		return nil, err
	}
	t.cfg.OnProgress(float64(out.Len()))
	t.cfg.OnMessage(fmt.Sprintf("Done, took: %s", time.Since(start).Round(time.Second).String()))

	return out.Bytes(), nil
}

func (t *Client) readMemoryByAddress(ctx context.Context, address, length int) ([]byte, error) {
	// Jump to read adress
	t.c.SendFrame(0x240, []byte{0x41, 0xA1, 0x08, 0x2C, 0xF0, 0x03, 0x00, byte(length)}, gocan.Outgoing)
	frame := gocan.NewFrame(0x240, []byte{0x00, 0xA1, byte((address >> 16) & 0xFF), byte((address >> 8) & 0xFF), byte(address & 0xFF), 0x00, 0x00, 0x00}, gocan.ResponseRequired)
	f, err := t.c.SendAndWait(ctx, frame, t.defaultTimeout*3, 0x258)
	if err != nil {
		return nil, err
	}
	d := f.Data()
	t.Ack(d[0], gocan.Outgoing)

	if d[3] != 0x6C || d[4] != 0xF0 {
		return nil, fmt.Errorf("failed to jump to 0x%X got response: %s", address, f.String())
	}
	b, err := t.recvData(ctx, length)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (t *Client) recvData(ctx context.Context, length int) ([]byte, error) {
	var receivedBytes, payloadLeft int
	out := bytes.NewBuffer([]byte{})

	sub := t.c.Subscribe(ctx, 0x258)
	defer sub.Close()

	startTransfer := gocan.NewFrame(0x240, []byte{0x40, 0xA1, 0x02, 0x21, 0xF0, 0x00, 0x00, 0x00}, gocan.ResponseRequired)
	if err := t.c.Send(startTransfer); err != nil {
		return nil, err
	}

outer:
	for receivedBytes < length {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(t.defaultTimeout * 4):
			return nil, fmt.Errorf("timeout")

		case f := <-sub.Chan():
			d := f.Data()
			if d[0]&0x40 == 0x40 {
				payloadLeft = int(d[2]) - 2 // subtract two non-payload bytes

				if payloadLeft > 0 && receivedBytes < length {
					out.WriteByte(d[5])
					receivedBytes++
					payloadLeft--
				}
				if payloadLeft > 0 && receivedBytes < length {
					out.WriteByte(d[6])
					receivedBytes++
					payloadLeft--
				}
				if payloadLeft > 0 && receivedBytes < length {
					out.WriteByte(d[7])
					receivedBytes++
					payloadLeft--
				}
			} else {
				for i := 0; i < 6; i++ {
					if receivedBytes < length {
						out.WriteByte(d[2+i])
						receivedBytes++
						payloadLeft--
						if payloadLeft == 0 {
							break
						}
					}
				}
			}
			if d[0] == 0x80 || d[0] == 0xC0 {
				t.Ack(d[0], gocan.Outgoing)
				break outer
			} else {
				t.Ack(d[0], gocan.ResponseRequired)
			}
		}
	}
	return out.Bytes(), nil
}

func (t *Client) endDownloadMode(ctx context.Context) error {
	frameData := gocan.NewFrame(0x240, []byte{0x40, 0xA1, 0x01, 0x82, 0x00, 0x00, 0x00, 0x00}, gocan.ResponseRequired)
	resp, err := t.c.SendAndWait(ctx, frameData, t.defaultTimeout, 0x258)
	if err != nil {
		return fmt.Errorf("end download mode: %v", err)
	}
	d := resp.Data()
	t.Ack(d[0], gocan.Outgoing)
	return nil
}
