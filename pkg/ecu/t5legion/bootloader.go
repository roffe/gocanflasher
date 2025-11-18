package t5legion

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/roffe/gocan"
)

func (t *Client) Ping(ctx context.Context) error {
	frame := gocan.NewFrame(0x05, []byte{0xEF, 0xBE, 0x00, 0x00, 0x00, 0x00, 0x33, 0x66}, gocan.ResponseRequired)
	resp, err := t.c.SendAndWait(ctx, frame, t.defaultTimeout, 0x0C)
	if err != nil {
		return errors.New("LegionPing: " + err.Error())
	}
	if resp.Data[0] == 0xDE && resp.Data[1] == 0xAD && resp.Data[2] == 0xF0 && resp.Data[3] == 0x0F {
		return nil
	}
	return errors.New("LegionPing: no response")
}

func (t *Client) UploadBootLoader(ctx context.Context) error {
	err := retry.Do(
		func() error {
			return t.Ping(ctx)
		},
		retry.Attempts(3),
		retry.Delay(100*time.Millisecond),
		retry.OnRetry(func(n uint, err error) {
			t.cfg.OnError(err)
		}),
	)
	if err == nil {
		return nil
	}

	start := time.Now()
	t.cfg.OnProgress(-float64(len(LegionBootloader)))
	t.cfg.OnMessage("Uploading legion bootloader...")
	var progress float64 = 0
	r := bytes.NewReader(LegionBootloader)
	for r.Len() > 0 {
		payload := make([]byte, 8)
		n, err := r.Read(payload)
		if err != nil {
			return err
		}
		if n != 8 {
			return fmt.Errorf("invalid read length: %d", n)
		}

		switch payload[0] {
		case 0xA5:
			f, err := t.c.SendAndWait(
				ctx,
				gocan.NewFrame(0x5, payload, gocan.ResponseRequired),
				250*time.Millisecond,
				0xC,
			)
			if err != nil {
				return fmt.Errorf("failed to sendBootloaderAddressCommand: %v", err)
			}
			if f.DLC() != 8 || f.Data[0] != 0xA5 || f.Data[1] != 0x00 {
				return fmt.Errorf("invalid response to sendBootloaderAddressCommand")
			}
		case 0xC1:
			f, err := t.c.SendAndWait(
				ctx,
				gocan.NewFrame(0x5, payload, gocan.ResponseRequired),
				250*time.Millisecond,
				0xC,
			)
			if err != nil {
				return err
			}
			if f.Data[0] != 0x0E || f.Data[1] != 0x00 {
				return fmt.Errorf("invalid response to sendBootloaderDataCommand: %X", f.Data)
			}
		default:
			resp, err := t.c.SendAndWait(
				ctx,
				gocan.NewFrame(0x5, payload, gocan.ResponseRequired),
				150*time.Millisecond,
				0xC,
			)
			if err != nil {
				return err
			}
			if resp.Data[0] == 0x1C && resp.Data[1] == 0x01 && resp.Data[2] == 0x00 {
				t.cfg.OnMessage("Bootloader already running")
				t.bootloaded = true
				return nil
			}

			if resp.DLC() != 8 || resp.Data[1] != 0x00 {
				return fmt.Errorf("failed to upload bootloader: %X", resp.Data)
			}
		}

		progress += 8
		t.cfg.OnProgress(progress)
	}
	t.cfg.OnMessage(fmt.Sprintf("Done, took: %s", time.Since(start).Round(time.Millisecond).String()))
	t.bootloaded = true
	return nil
}

func (t *Client) readDataByLocalIdentifier(ctx context.Context, pci byte, address int, length byte) ([]byte, int, error) {
	retData := make([]byte, length)
	payload := []byte{pci, 0x21, length, byte(address >> 24), byte(address >> 16), byte(address >> 8), byte(address), 0x00}
	frame := gocan.NewFrame(0x05, payload, gocan.ResponseRequired)
	//log.Println(frame.ColorString())
	resp, err := t.c.SendAndWait(ctx, frame, t.defaultTimeout, 0x0C)
	if err != nil {
		return nil, 0, err
	}
	//log.Println(resp.(*gocan.Frame).ColorString())

	if resp.Data[0] == 0x7E {
		return nil, 0, errors.New("got 0x7E message as response to 0x21, ReadDataByLocalIdentifier command")
	}

	if resp.Data[3] > 0 {
		retData[0] = 0xFF
		for j := 1; j < len(retData); j *= 2 {
			copy(retData[j:], retData[:j])
		}
		return retData, int(resp.Data[3]), nil
	}

	rx_cnt := 0
	copy(retData, resp.Data[4:])
	rx_cnt += 4

	sub := t.c.Subscribe(ctx, 0x0C)
	defer sub.Close()

	if err := t.c.Send(0x05, []byte{0x30}, gocan.CANFrameType{Type: 2, Responses: 19}); err != nil {
		return nil, 0, err
	}

	var seq byte = 0x21
	for rx_cnt < int(length) {
		select {
		case resp := <-sub.Chan():
			if resp.Data[0] != seq {
				return nil, 0, fmt.Errorf("got unexpected sequence number %02X, expected %02X", resp.Data[0], seq)
			}
			copy(retData[rx_cnt:], resp.Data[1:])
			rx_cnt += 7
			seq++
			if seq > 0x2F {
				seq = 0x20
			}
		case <-time.After(1 * time.Second):
			return nil, 0, errors.New("timeout")
		}

	}

	return retData, 0, nil
}

func (t *Client) sendBootloaderAddressCommand(ctx context.Context, address uint32, len byte) error {
	payload := []byte{0xA5, byte(address >> 24), byte(address >> 16), byte(address >> 8), byte(address), len, 0x00, 0x00}
	f, err := t.c.SendAndWait(
		ctx,
		gocan.NewFrame(0x5, payload, gocan.ResponseRequired),
		250*time.Millisecond,
		0xC,
	)
	if err != nil {
		return fmt.Errorf("failed to sendBootloaderAddressCommand: %v", err)
	}
	if f.DLC() != 8 || f.Data[0] != 0xA5 || f.Data[1] != 0x00 {
		return fmt.Errorf("invalid response to sendBootloaderAddressCommand")
	}
	return nil
}

/*
func (t *Client) sendBootVectorAddressSRAM(address uint32) error {
	data := []byte{0xC1, byte(address >> 24), byte(address >> 16), byte(address >> 8), byte(address), 0x00, 0x00, 0x00}
	return t.c.SendFrame(0x5, data, gocan.Outgoing)
}
*/

func (t *Client) sendBootloaderDataCommand(ctx context.Context, data []byte, length byte) error {
	frame := gocan.NewFrame(0x5, data, gocan.ResponseRequired)
	resp, err := t.c.SendAndWait(ctx, frame, 150*time.Millisecond, 0xC)
	if err != nil {
		return fmt.Errorf("failed SBLDC: %v", err)
	}
	if resp.Data[1] != 0x00 {
		return errors.New("failed to write")
	}
	return nil
}

const (
	SetInterFrameLatency      Command = 0x00
	GetCRC32                  Command = 0x01
	GetTrionic5MD5            Command = 0x02
	RetrieveSystemInformation Command = 0x06
)

type Command byte

func (c Command) String() string {
	switch c {
	case SetInterFrameLatency:
		return "configure inter frame latency"
	case GetCRC32:
		return "full crc-32"
	case GetTrionic5MD5:
		return "Trionic8 MD5"
	case RetrieveSystemInformation:
		return "Retrieve system information"
	default:
		return "unknown command"
	}
}

type SystemInformation struct {
	FirmwareBase uint32
	FlashType    byte
	FlashSize    uint32
	FlashID      uint16
}

func (t *Client) RetrieveSystemInformation(ctx context.Context) (*SystemInformation, error) {
	buf, err := t.IDemand(ctx, RetrieveSystemInformation, 0)
	if err != nil {
		return nil, err
	}

	flash_type := (buf[0] & 0xF0) >> 4
	flash_base := uint32(buf[0]&0x0F) << 16
	flash_size := uint32(buf[1]&0x0F) << 16
	flash_id := uint16(buf[2])<<8 | uint16(buf[3])

	return &SystemInformation{
		FirmwareBase: flash_base,
		FlashType:    flash_type,
		FlashSize:    flash_size,
		FlashID:      flash_id,
	}, nil

}

func (t *Client) IDemand(ctx context.Context, command Command, wish uint16) ([]byte, error) {
	//log.Println("Legion i demand", command.String()+"!")
	payload := []byte{0x02, 0xA5, byte(command), 0x00, 0x00, 0x00, byte(wish >> 8), byte(wish)}
	frame := gocan.NewFrame(0x5, payload, gocan.ResponseRequired)

	var out []byte

	err := retry.Do(func() error {
		resp, err := t.c.SendAndWait(ctx, frame, t.defaultTimeout, 0xC)
		if err != nil {
			return err
		}
		if err := demandErr(command, resp.Data); err != nil {
			return err
		}
		switch command {
		case 0:
			// Settings correctly received.
			//log.Println("settings correctly received")
			return nil
		case 1:
			// Crc-32; complete
			//log.Println("crc-32 complete")
			out = resp.Data[4:]
			return nil
		case 2, 3:
			// md5; complete
			//b, _, err := t.ReadDataByLocalIdentifier(ctx, true, 7, 0, 16)
			//if err != nil {
			//	return err
			//}
			//out = b
			log.Println("md5...")
			return nil
		case 4:
			return nil
		case 5:
			return nil
		case 6:
			// Retrieve system information
			out = resp.Data[4:]
			return nil
		default:
			return retry.Unrecoverable(errors.New("command unknown to the legion"))
		}
	},
		retry.Context(ctx),
		retry.Attempts(10),
		retry.LastErrorOnly(true),
		//retry.OnRetry(func(n uint, err error) {
		//	log.Printf("#%d: %v", n, err)
		//}),
	)
	if err != nil {
		return out, err
	}

	return out, nil
}

func demandErr(command Command, d []byte) error {
	if command == 0 && d[3] != 1 {
		return errors.New("failed to set inter frame latency")
	}

	if command == 1 && d[3] != 1 {
		time.Sleep(150 * time.Millisecond)
		return errors.New("crc-32 not ready yet :(")
	}

	if (command == 2 || command == 3) && d[3] != 1 {
		time.Sleep(150 * time.Millisecond)
		return errors.New("MD5 not ready yet :(")
	}

	if command == 4 && d[3] != 0x01 {
		time.Sleep(100 * time.Millisecond)
		return errors.New("secondary loader not ready yet")
	}

	if command == 5 {
		switch d[3] {
		case 0xFF:
			// Critical error; Could not start the secondary loader!
			return errors.New("failed to start the secondary loader")
		case 0xFD:
			// Failed to write!
			return errors.New("retrying write")
		case 0xFE:
			// Failed to format!
			return errors.New("retrying format")
		default:
			time.Sleep(500 * time.Millisecond)
			return errors.New("busy marrying")
		}

	}

	if command == 6 && d[3] != 0x01 {
		return fmt.Errorf("ADC read not ready yet")
	}

	// Something is wrong or we sent the wrong command.
	if d[3] == 0xFF {
		return errors.New("bootloader did what it could and failed. Sorry")

	}
	return nil
}
