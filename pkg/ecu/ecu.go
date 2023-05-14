package ecu

import (
	"context"
	"errors"
	"log"
	"sort"

	"github.com/roffe/gocan"
	"github.com/roffe/gocanflasher/pkg/model"
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

type Config struct {
	Name       string
	OnProgress func(float64)
	OnError    func(error)
	OnMessage  func(string)
}

func LoadConfig(cfg *Config) *Config {
	if cfg == nil {
		cfg = &Config{
			Name: "Unknown ECU",
		}
	}

	if cfg.OnProgress == nil {
		cfg.OnProgress = func(f float64) {
			log.Println(f)
		}
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

var ecuMap = map[string]*EcuInfo{}

type EcuInfo struct {
	Name    string
	NewFunc func(c *gocan.Client, cfg *Config) Client
	CANRate float64
	Filter  []uint32
}

func Register(t *EcuInfo) {
	if _, found := ecuMap[t.Name]; found {
		panic("ECU already registered: " + t.Name)
	}
	ecuMap[t.Name] = t
}

func New(c *gocan.Client, cfg *Config) (Client, error) {
	if ecu, found := ecuMap[cfg.Name]; found {
		return ecu.NewFunc(c, cfg), nil
	}
	return nil, errors.New("unknown ECU")
}

func List() (ecus []string) {
	for k := range ecuMap {
		ecus = append(ecus, k)
	}
	sort.Strings(ecus)
	return
}

func Filters(ecuName string) []uint32 {
	e, found := ecuMap[ecuName]
	if !found {
		return []uint32{}
	}
	return e.Filter
}

func CANRate(ecuName string) float64 {
	e, found := ecuMap[ecuName]
	if !found {
		return 0
	}
	return e.CANRate
}
