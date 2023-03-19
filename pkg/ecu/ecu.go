package ecu

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/roffe/gocan"
	"github.com/roffe/gocanflasher/pkg/model"
)

const (
	UnknownECU Type = -1
	Trionic5   Type = iota
	Trionic7
	Trionic8
	Trionic8MCP
	Me96
)

type Client interface {
	ReadDTC(context.Context) ([]model.DTC, error)
	PrintECUInfo(context.Context) error
	Info(context.Context) ([]model.HeaderResult, error)
	DumpECU(context.Context) ([]byte, error)
	FlashECU(context.Context, []byte) error
	EraseECU(context.Context) error
	ResetECU(context.Context) error
}

func FromString(s string) Type {
	switch strings.ToLower(s) {
	case "5", "t5", "trionic5", "trionic 5":
		return Trionic5
	case "7", "t7", "trionic7", "trionic 7":
		return Trionic7
	case "8", "t8", "trionic8", "trionic 8":
		return Trionic8
	case "mcp", "t8mcp", "trionic8mcp", "trionic 8 mcp":
		return Trionic8MCP
	case "96", "me9.6", "me96", "me 9.6":
		return Me96
	default:
		return UnknownECU
	}
}

type Type int

func (e Type) String() string {
	switch e {
	case Trionic5:
		return "Trionic 5"
	case Trionic7:
		return "Trionic 7"
	case Trionic8:
		return "Trionic 8"
	case Trionic8MCP:
		return "Trionic 8 MCP"
	case Me96:
		return "Me 9.6"
	default:
		return "Unknown ECU"
	}
}

type Config struct {
	Type       Type
	OnProgress model.ProgressCallback
	OnError    func(error)
	OnMessage  func(string)
}

func onProgress(v interface{}) {
	log.Println(v)
}

func LoadConfig(cfg *Config) *Config {
	if cfg == nil {
		cfg = &Config{
			Type: UnknownECU,
		}
	}

	if cfg.OnProgress == nil {
		cfg.OnProgress = onProgress
	}

	if cfg.OnError == nil {
		cfg.OnError = func(err error) {
			log.Println(err)
		}
	}

	if cfg.OnMessage == nil {
		cfg.OnMessage = func(msg string) {
			log.Println(msg)
		}
	}

	return cfg
}

var ecuMap = map[Type]func(c *gocan.Client, cfg *Config) Client{}

func Register(t Type, f func(c *gocan.Client, cfg *Config) Client) {
	ecuMap[t] = f
}

func New(c *gocan.Client, cfg *Config) (Client, error) {
	if ecu, found := ecuMap[cfg.Type]; found {
		return ecu(c, cfg), nil
	}
	return nil, errors.New("unknown ECU")
	/*
		switch cfg.Type {
		case Trionic5:
			return t5.New(c, cfg), nil
		case Trionic7:
			return t7.New(c, cfg), nil
		case Trionic8:
			return t8.New(c, cfg), nil
		case Trionic8MCP:
			return t8mcp.New(c, cfg), nil
		default:
			return nil, errors.New("unknown ECU")
		}
	*/
}

func CANFilters(t Type) []uint32 {
	switch t {
	case Trionic5:
		return []uint32{0x0, 0x05, 0x06, 0x0C}
	case Trionic7:
		return []uint32{0x220, 0x238, 0x240, 0x258, 0x266}
	case Trionic8:
		return []uint32{0x5E8, 0x7E8}
	case Trionic8MCP:
		return []uint32{0x7E8}
	default:
		return []uint32{}
	}
}
