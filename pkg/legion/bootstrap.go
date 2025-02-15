package legion

import (
	"context"
	"time"

	"github.com/avast/retry-go"
	"github.com/roffe/gocan/pkg/gmlan"
	"github.com/roffe/gocanflasher/pkg/ecu/t8sec"
)

// Disable normal communication, enter programming mode, and request security access
// then upload bootloader and jump to it

func (t *Client) Alive(ctx context.Context) bool {

	t.cfg.OnMessage("Checking if Legion is running")

	err := retry.Do(func() error {
		err := t.Ping(ctx)
		if err != nil {
			return err
		}
		t.cfg.OnMessage("Legion is ready")
		t.legionRunning = true
		return nil
	},
		retry.Attempts(3),
		retry.Context(ctx),
		retry.LastErrorOnly(true),
	)
	return err == nil
}

func (t *Client) Bootstrap(ctx context.Context) error {
	if !t.Alive(ctx) {
		if err := t.bootstrapPreFlight(ctx); err != nil {
			return err
		}
		if err := t.UploadBootloader(ctx); err != nil {
			return err
		}

		t.cfg.OnMessage("Starting bootloader")

		if err := t.StartBootloader(ctx, 0x102400); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
		t.legionRunning = t.Alive(ctx)
	}

	if t.legionRunning {
		t.cfg.OnMessage("Legion is running, enabling high speed mode")
		if err := t.EnableHighSpeed(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (t *Client) bootstrapPreFlight(ctx context.Context) error {
	t.gm.TesterPresentNoResponseAllowed()

	//time.Sleep(50 * time.Millisecond)

	if err := t.gm.InitiateDiagnosticOperation(ctx, 0x02); err != nil {
		return err
	}

	if err := t.gm.DisableNormalCommunication(ctx); err != nil {
		return err
	}

	if b, err := t.gm.ReportProgrammedState(ctx); err != nil {
		return err
	} else {
		t.cfg.OnMessage("ECU Programmed state: " + gmlan.TranslateProgrammedState(b))
	}

	if err := t.gm.ProgrammingModeRequest(ctx); err != nil {
		return err
	}

	if err := t.gm.ProgrammingModeEnable(ctx); err != nil {
		return err
	}

	time.Sleep(50 * time.Millisecond)

	t.gm.TesterPresentNoResponseAllowed()

	t.cfg.OnMessage("Requesting t8 security access")

	if err := t.gm.RequestSecurityAccess(ctx, 0x01, 0, t8sec.CalculateAccessKey); err != nil {
		return err
	}

	time.Sleep(50 * time.Millisecond)
	return nil
}
