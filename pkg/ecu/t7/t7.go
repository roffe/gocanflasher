package t7

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/roffe/gocan"
	"github.com/roffe/gocanflasher/pkg/ecu"
)

func init() {
	ecu.Register(&ecu.EcuInfo{
		Name:    "Trionic 7",
		NewFunc: New,
		CANRate: 500,
		Filter:  []uint32{0x238, 0x258, 0x266},
	})
}

type Client struct {
	c              *gocan.Client
	defaultTimeout time.Duration
	cfg            *ecu.Config
}

func New(c *gocan.Client, cfg *ecu.Config) ecu.Client {
	t := &Client{
		c:              c,
		cfg:            ecu.LoadConfig(cfg),
		defaultTimeout: 250 * time.Millisecond,
	}
	return t
}

// 266h Send acknowledgement, has 0x3F on 3rd!
func (t *Client) Ack(val byte, typ gocan.CANFrameType) error {
	return t.c.Send(0x266, []byte{0x40, 0xA1, 0x3F, val & 0xBF, 0x00, 0x00, 0x00, 0x00}, typ)
}

var lastDataInitialization time.Time

func (t *Client) DataInitialization(ctx context.Context) error {
	if !lastDataInitialization.IsZero() {
		if time.Since(lastDataInitialization) < 8*time.Second {
			return nil
		}
	}
	lastDataInitialization = time.Now()

	err := retry.Do(
		func() error {
			resp, err := t.c.SendAndWait(ctx, gocan.NewFrame(0x220, []byte{0x3F, 0x81, 0x00, 0x11, 0x02, 0x40, 0x00, 0x00}, gocan.ResponseRequired), t.defaultTimeout, 0x238)
			if err != nil {
				return fmt.Errorf("%v", err)
			}
			if !bytes.Equal(resp.Data, []byte{0x40, 0xBF, 0x21, 0xC1, 0x00, 0x11, 0x02, 0x58}) {
				return fmt.Errorf("/!\\ Invalid data initialization response")
			}

			return nil
		},
		retry.Context(ctx),
		retry.Attempts(6),
		retry.OnRetry(func(n uint, err error) {
			if n == 0 {
				t.cfg.OnError(err)
				return
			}
			t.cfg.OnMessage(fmt.Sprintf("Retry #%d, %v", n, err))
		}),
		retry.LastErrorOnly(true),
		retry.Delay(250*time.Millisecond),
	)
	if err != nil {
		return fmt.Errorf("/!\\ Data initialization failed: %v", err)
	}
	return nil
}

func (t *Client) GetHeader(ctx context.Context, id byte) (string, error) {
	err := retry.Do(
		func() error {
			return t.c.Send(0x240, []byte{0x40, 0xA1, 0x02, 0x1A, id, 0x00, 0x00, 0x00}, gocan.ResponseRequired)
		},
		retry.Context(ctx),
		retry.Attempts(3),
	)
	if err != nil {
		return "", fmt.Errorf("failed getting header: %v", err)
	}

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("failed getting header: %v", err)
	default:
	}

	var answer []byte
	var length int
	for i := 0; i < 10; i++ {
		f, err := t.c.Wait(ctx, t.defaultTimeout, 0x258)
		if err != nil {
			return "", err
		}
		if f.Data[0]&0x40 == 0x40 {
			if int(f.Data[2]) > 2 {
				length = int(f.Data[2]) - 2
			}
			for b := 5; b < 8; b++ {
				if length > 0 {
					answer = append(answer, f.Data[b])
				}
				length--
			}
		} else {
			for c := 0; c < 6; c++ {
				if length == 0 {
					break
				}
				answer = append(answer, f.Data[2+c])
				length--
			}
		}

		if f.Data[0] == 0x80 || f.Data[0] == 0xC0 {
			t.Ack(f.Data[0], gocan.Outgoing)
			break
		} else {
			t.Ack(f.Data[0], gocan.ResponseRequired)
		}
	}

	return string(answer), nil
}

func (t *Client) KnockKnock(ctx context.Context) (bool, error) {
	if err := t.DataInitialization(ctx); err != nil {
		return false, err
	}
	for i := 0; i <= 4; i++ {
		ok, err := t.letMeIn(ctx, i)
		if err != nil {
			t.cfg.OnError(fmt.Errorf("/!\\ Failed to obtain security access: %v", err))
			time.Sleep(3 * time.Second)
			continue

		}
		if ok {
			t.cfg.OnMessage("Security access obtained")
			return true, nil
		}
	}
	return false, fmt.Errorf("/!\\ Failed to obtain security access")
}

func (t *Client) letMeIn(ctx context.Context, method int) (bool, error) {
	msg := []byte{0x40, 0xA1, 0x02, 0x27, 0x05, 0x00, 0x00, 0x00}
	msgReply := []byte{0x40, 0xA1, 0x04, 0x27, 0x06, 0x00, 0x00, 0x00}

	f, err := t.c.SendAndWait(ctx, gocan.NewFrame(0x240, msg, gocan.ResponseRequired), t.defaultTimeout, 0x258)
	if err != nil {
		return false, fmt.Errorf("request seed: %v", err)

	}
	t.Ack(f.Data[0], gocan.ResponseRequired)

	s := int(f.Data[5])<<8 | int(f.Data[6])
	k := calcen(s, method)

	msgReply[5] = byte(int(k) >> 8 & int(0xFF))
	msgReply[6] = byte(k) & 0xFF

	f2, err := t.c.SendAndWait(ctx, gocan.NewFrame(0x240, msgReply, gocan.ResponseRequired), t.defaultTimeout, 0x258)
	if err != nil {
		return false, fmt.Errorf("send seed: %v", err)

	}
	t.Ack(f2.Data[0], gocan.ResponseRequired)
	if f2.Data[3] == 0x67 && f2.Data[5] == 0x34 {
		return true, nil
	} else {
		log.Println(f2.String())
		return false, errors.New("invalid response")
	}
}

func calcen(seed int, method int) int {
	key := seed << 2
	key &= 0xFFFF
	switch method {
	case 0:
		key ^= 0x8142
		key -= 0x2356
	case 1:
		key ^= 0x4081
		key -= 0x1F6F
	case 2:
		key ^= 0x3DC
		key -= 0x2356
	case 3:
		key ^= 0x3D7
		key -= 0x2356
	case 4:
		key ^= 0x409
		key -= 0x2356
	}
	key &= 0xFFFF
	return key
}

func (t *Client) LetMeTry(ctx context.Context, key1, key2 int) bool {
	msg := []byte{0x40, 0xA1, 0x02, 0x27, 0x05, 0x00, 0x00, 0x00}
	msgReply := []byte{0x40, 0xA1, 0x04, 0x27, 0x06, 0x00, 0x00, 0x00}

	f, err := t.c.SendAndWait(ctx, gocan.NewFrame(0x240, msg, gocan.ResponseRequired), t.defaultTimeout, 0x258)
	if err != nil {
		log.Println(err)
		return false

	}
	t.Ack(f.Data[0], gocan.ResponseRequired)

	s := int(f.Data[5])<<8 | int(f.Data[6])
	k := calcenCustom(s, key1, key2)

	msgReply[5] = byte(int(k) >> 8 & int(0xFF))
	msgReply[6] = byte(k) & 0xFF

	f2, err := t.c.SendAndWait(ctx, gocan.NewFrame(0x240, msgReply, gocan.ResponseRequired), t.defaultTimeout, 0x258)
	if err != nil {
		log.Println(err)
		return false

	}

	t.Ack(f2.Data[0], gocan.ResponseRequired)

	if f2.Data[3] == 0x67 && f2.Data[5] == 0x34 {
		return true
	} else {
		return false
	}
}

func calcenCustom(seed int, key1, key2 int) int {
	key := seed << 2
	key &= 0xFFFF
	key ^= key1
	key -= key2
	key &= 0xFFFF
	return key
}

func (t *Client) StopSession(ctx context.Context) error {
	return t.c.Send(0x220, []byte{0x40, 0xA1, 0x02, 0x82, 0x00, 0x00, 0x00, 0x00}, gocan.ResponseRequired)
}
