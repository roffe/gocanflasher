package t8mcp

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"time"

	"github.com/roffe/gocan"
	"github.com/roffe/gocanflasher/pkg/ecu"
	"github.com/roffe/gocanflasher/pkg/ecu/t8util"
	"github.com/roffe/gocanflasher/pkg/model"
	"github.com/roffe/gocanflasher/pkg/t8legion"
)

func init() {
	ecu.Register(&ecu.EcuInfo{
		Name:    "Trionic 8 MCP",
		NewFunc: New,
		CANRate: 500,
		Filter:  []uint32{0x7E8},
	})
}

type Client struct {
	c              *gocan.Client
	cfg            *ecu.Config
	defaultTimeout time.Duration
	legion         *t8legion.Client
}

func New(c *gocan.Client, cfg *ecu.Config) ecu.Client {
	t := &Client{
		c:              c,
		cfg:            ecu.LoadConfig(cfg),
		defaultTimeout: 150 * time.Millisecond,
		legion:         t8legion.New(c, cfg, 0x7e0, 0x7e8),
	}
	return t
}

func (t *Client) ReadDTC(ctx context.Context) ([]model.DTC, error) {
	return nil, errors.New("MCP cannot do this")
}

func (t *Client) Info(ctx context.Context) ([]model.HeaderResult, error) {
	if err := t.legion.Bootstrap(ctx); err != nil {
		return nil, err
	}

	_, err := t.legion.IDemand(ctx, t8legion.StartSecondaryBootloader, 0)
	if err != nil {
		return nil, errors.New("failed to start secondary bootloader")
	}

	ver, err := t.legion.GetMCPVersion(ctx)
	if err != nil {
		return nil, err
	}

	t.cfg.OnMessage("MCP Firmware information: " + ver)

	return nil, nil
}

func (t *Client) PrintECUInfo(ctx context.Context) error {
	return nil
}

func (t *Client) FlashECU(ctx context.Context, bin []byte) error {

	if err := t.legion.Bootstrap(ctx); err != nil {
		return err
	}
	t.cfg.OnMessage("Comparing MD5's for erase")
	t.cfg.OnProgress(-9)
	t.cfg.OnProgress(0)
	for i := 1; i <= 9; i++ {
		lmd5 := t8util.GetPartitionMD5(bin, 6, i)
		md5, err := t.legion.GetMD5(ctx, t8legion.GetTrionic8MCPMD5, uint16(i))
		if err != nil {
			return err
		}
		t.cfg.OnMessage(fmt.Sprintf("local partition   %d> %X", i, lmd5))
		t.cfg.OnMessage(fmt.Sprintf("remote partition  %d> %X", i, md5))
		t.cfg.OnProgress(float64(i))
	}

	return nil
}

func (t *Client) DumpECU(ctx context.Context) ([]byte, error) {
	if err := t.legion.Bootstrap(ctx); err != nil {
		return nil, err
	}

	_, err := t.legion.IDemand(ctx, t8legion.StartSecondaryBootloader, 0)
	if err != nil {
		return nil, errors.New("failed to start secondary bootloader")
	}
	t.cfg.OnMessage("Dumping MCP")

	start := time.Now()

	bin, err := t.legion.ReadFlash(ctx, t8legion.EcuByte_MCP, 0x40100, false)
	if err != nil {
		return nil, err
	}

	ecumd5bytes, err := t.legion.IDemand(ctx, t8legion.GetTrionic8MCPMD5, 0x00)
	if err != nil {
		return nil, err
	}
	calculatedMD5 := md5.Sum(bin)

	t.cfg.OnMessage(fmt.Sprintf("Remote md5 : %X", ecumd5bytes))
	t.cfg.OnMessage(fmt.Sprintf("Local md5  : %X", calculatedMD5))

	if !bytes.Equal(ecumd5bytes, calculatedMD5[:]) {
		return nil, errors.New("md5 Verification failed")
	}

	t.cfg.OnMessage("Done, took: " + time.Since(start).String())

	return bin, nil
}

func (t *Client) EraseECU(ctx context.Context) error {
	return nil
}

func (t *Client) ResetECU(ctx context.Context) error {
	if t.legion.IsRunning() {
		if err := t.legion.Exit(ctx); err != nil {
			return err
		}
	}
	return nil
}
