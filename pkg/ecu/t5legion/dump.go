package t5legion

import (
	"context"
	"fmt"
	"log"
	"time"
)

func (t *Client) DumpECU(ctx context.Context) ([]byte, error) {
	if !t.bootloaded {
		if err := t.UploadBootLoader(ctx); err != nil {
			return nil, err
		}
	}

	_, err := t.IDemand(ctx, SetInterFrameLatency, 80)
	if err != nil {
		return nil, err
	}

	ecutype, err := t.DetermineECU(ctx)
	if err != nil {
		return nil, err
	}

	start := getstartAddress(ecutype)
	length := 0x80000 - start
	blockSize := 0x80 // defined in bootloader... keep it that way!

	t.cfg.OnProgress(-float64(length))
	t.cfg.OnMessage("Dumping ECU")

	buffer := make([]byte, length)

	// Pre-fill buffer with 0xFF (unprogrammed FLASH chip value)
	buffer[0] = 0xFF
	for j := 1; j < len(buffer); j *= 2 {
		copy(buffer[j:], buffer[:j])
	}

	startTime := time.Now()
	progress := 0
	for uint32(progress) < length {
		d, blocksToSkip, err := t.readDataByLocalIdentifier(ctx, 0x06, int(start), 0x80)
		if err != nil {
			return nil, err
		}
		if blocksToSkip > 0 {
			log.Println("Skipping", blocksToSkip, "blocks")
			start += uint32(blocksToSkip * blockSize)
			continue
		}
		copy(buffer[progress:], d)
		start += uint32(blockSize)
		progress += blockSize
		t.cfg.OnProgress(float64(progress))
	}
	t.cfg.OnMessage(fmt.Sprintf("Dumping ECU done, took %s", time.Since(startTime)))
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
