package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/metno/go-mms/pkg/mms"
	"github.com/prometheus/client_golang/prometheus"
)

type Product struct {
	Name                 string
	NextInstanceExpected time.Time
}

type Productstatus struct {
	Products map[string]Product
	GaugeVec *prometheus.GaugeVec
	mu       sync.RWMutex
}

func NewProductstatus(m *metrics) *Productstatus {
	productstatus := Productstatus{
		Products: make(map[string]Product),
		GaugeVec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Subsystem: "mmsd",
				Name:      "product_delay",
				Help:      "Delay of a product in seconds.",
			},
			[]string{
				// The product represented as a label
				"product",
			},
		),
	}

	m.MustRegister(productstatus.GaugeVec)

	return &productstatus
}

func (p *Productstatus) PushEvent(pe mms.ProductEvent) error {
	if time.Time(pe.NextEventAt).Equal(time.Time(pe.CreatedAt)) {
		return nil
	}

	p.mu.Lock() // Used Lock() (not RLock()) because we're writing to the map
	defer p.mu.Unlock()

	p.Products[pe.Product] = Product{
		Name:                 pe.Product,
		NextInstanceExpected: time.Time(pe.NextEventAt),
	}
	return nil
}

func (p *Productstatus) GetProductDelays(t time.Time) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for k, v := range p.Products {
		diff := t.Sub(v.NextInstanceExpected)
		fmt.Printf("%s: %v\n", k, diff.Seconds())
	}
}

func (p *Productstatus) UpdateMetrics() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for k, v := range p.Products {
		diff := time.Now().Sub(v.NextInstanceExpected)
		p.GaugeVec.WithLabelValues(k).Set(diff.Seconds())
	}
}

func (p *Productstatus) Populate(events []*mms.ProductEvent) {
	for _, event := range events {
		p.PushEvent(*event)
	}
}
