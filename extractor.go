package main

import (
	"github.com/vitkovskii/insane-doc/logger"
	"regexp"
	"strconv"
	"strings"
)

type extractorType int

const (
	extractorTypeConst extractorType = iota
	extractorTypeSplit
	extractorTypeRegexp
)

type extractor struct {
	t    extractorType
	data string
}

func (e *extractor) extract(s string) string {
	switch e.t {
	case extractorTypeConst:
		return e.data

	case extractorTypeRegexp:
		r := regexp.MustCompile(e.data)
		matches := r.FindStringSubmatch(s)
		if len(matches) <= 1 {
			return ""
		}

		return matches[1]

	case extractorTypeSplit:
		field, err := strconv.Atoi(e.data)
		if err != nil {
			logger.Fatalf("field extractor has wrong field index: %s", e.data)
		}
		// translate from human to golang array index
		field--

		fields := strings.Fields(s)
		if field > len(fields)-1 {
			return ""
		}

		return fields[field]

	}
	logger.Fatalf("unknown extractor")
	return ""
}
