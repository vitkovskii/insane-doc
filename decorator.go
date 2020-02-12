package main

import (
	"fmt"
	"insane-doc/logger"
)

type decoratorType int

const (
	decoratorTypeNo decoratorType = iota
	decoratorTypePattern
)

type decorator struct {
	t    decoratorType
	data string
}

func NewDecorator(s string) *decorator {
	if len(s) == 0 {
		logger.Fatalf("empty extractor")
	}

	if s[0] == '_' {
		return &decorator{t: decoratorTypeNo}
	}

	if s[0] == '/' {
		if s[len(s)-1] != '/' {
			logger.Fatalf(`wrong pattern decorator`, s)
		}

		return &decorator{t: decoratorTypePattern, data: s[1 : len(s)-1]}
	}

	if s == "code" {
		return &decorator{t: decoratorTypePattern, data: "`%s`"}
	}

	logger.Fatalf("unknown decorator %s", s)
	return nil
}

func (d *decorator) decorate(s string) string {
	if s == "" {
		return s
	}

	switch d.t {
	case decoratorTypePattern:
		return fmt.Sprintf(d.data, s)
	case decoratorTypeNo:
		return s
	}
	logger.Fatalf("unknown decorator %d", d.t)
	return ""
}
