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
