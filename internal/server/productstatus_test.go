package server

import (
	"testing"
	"time"

	"github.com/metno/go-mms/pkg/mms"
)

func TestPushEvent(t *testing.T) {
	metrics := NewServiceMetrics(MetricsOpts{})
	ps := NewProductstatus(metrics)

	var productEventList [3]mms.ProductEvent

	productEventList[0] = mms.ProductEvent{
		Product:         "test",
		ProductLocation: "/dev/test",
		CreatedAt:       mms.PEventTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
		NextEventAt:     mms.PEventTime(time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)),
	}

	productEventList[1] = mms.ProductEvent{
		Product:         "test_2",
		ProductLocation: "/dev/test",
		CreatedAt:       mms.PEventTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
		NextEventAt:     mms.PEventTime(time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)),
	}

	productEventList[2] = mms.ProductEvent{
		Product:         "test",
		ProductLocation: "/dev/test",
		CreatedAt:       mms.PEventTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
		NextEventAt:     mms.PEventTime(time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)),
	}

	for _, pe := range productEventList {
		err := ps.PushEvent(pe)
		if err != nil {
			t.Errorf("failed to parse event: %s", err)
		}
	}

	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	ps.GetProductDelays(ts)

	ts = time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	ps.GetProductDelays(ts)

	ts = time.Date(2020, 1, 1, 14, 0, 0, 0, time.UTC)
	ps.GetProductDelays(ts)

}
