package printer

import (
	"bytes"
	"fmt"

	"github.com/robotomize/go-printenv/internal/analysis"
)

type Printer interface {
	Print() []byte
}

type Option func(*Options)

type Options struct {
	groupByPkg    bool
	groupByModule bool
	noDefault     bool
}

func WithGroup() Option {
	return func(p *Options) {
		p.groupByPkg = true
	}
}

func WithoutDefault() Option {
	return func(options *Options) {
		options.noDefault = true
	}
}

func WithGroupByModule() Option {
	return func(options *Options) {
		options.groupByModule = true
	}
}

func New(items []analysis.OutputEntry, opts ...Option) Printer {
	p := &printer{items: items}

	for _, o := range opts {
		o(&p.opts)
	}

	return p
}

type printer struct {
	opts  Options
	items []analysis.OutputEntry
}

func (p *printer) Print() []byte {
	buf := bytes.NewBufferString("")
	for _, e := range p.items {
		buf.WriteString(fmt.Sprintf("%s\n", e.String()))
	}

	return buf.Bytes()
}
