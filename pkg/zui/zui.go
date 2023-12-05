package zui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gizak/termui/v3/widgets"
	"github.com/threefoldtech/zos/pkg"
)

type module struct {
	render    Signaler
	labels    []labelData
	paragraph *widgets.Paragraph
	mu        *sync.Mutex
}

// Signaler interface to signal ZUI to render some element.
type Signaler interface {
	Signal()
}

type labelData struct {
	label  string
	errors []string
}

// New returns a new ZUI module.
func New(ctx context.Context, p *widgets.Paragraph, render Signaler) pkg.ZUI {
	zuiModule := &module{
		render:    render,
		labels:    make([]labelData, 0),
		paragraph: p,
		mu:        &sync.Mutex{},
	}
	go zuiModule.renderErrors(ctx)
	return zuiModule
}

var _ pkg.ZUI = (*module)(nil)

// PushErrors pushes the given errors to the ZUI module to be displayed.
// It can also remove stop displaying certain label by sending an empty errors slice.
func (m *module) PushErrors(label string, errors []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, data := range m.labels {
		if data.label == label {
			m.labels[i].errors = errors
			return nil
		}
	}
	m.labels = append(m.labels, labelData{label, errors})
	return nil
}

func (m *module) renderErrors(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
			m.mu.Lock()
			labels := make([]labelData, len(m.labels))
			copy(labels, m.labels)
			m.mu.Unlock()
			display(ctx, labels, m.paragraph, m.render)
		}
	}
}

func display(ctx context.Context, labels []labelData, p *widgets.Paragraph, render Signaler) {
	p.Text = "[No Errors!](fg:green)"

	for _, label := range labels {
		for _, e := range label.errors {
			p.Text = fmt.Sprintf("%s: [%s](fg:red)", label.label, e)
			render.Signal()
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}

	render.Signal()
}
