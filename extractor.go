package main

import (
	"insane-doc/logger"
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

func NewExtractor(s string) *extractor {
	if len(s) == 0 {
		logger.Fatalf("empty extractor")
	}

	if s[0] == '#' {
		return &extractor{t: extractorTypeSplit, data: s[1:]}
	}

	if s[0] == '/' {
		if s[len(s)-1] != '/' {
			logger.Fatalf(`wrong regexp extractor, no finish "/": %s`, s)
		}

		return &extractor{t: extractorTypeRegexp, data: s[1 : len(s)-1]}
	}

	if s[0] == '_' {
		return &extractor{t: extractorTypeConst, data: ""}
	}

	return &extractor{t: extractorTypeConst, data: s[:]}
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
			logger.Fatal("field extractor has wrong field index: %s", e.data)
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
