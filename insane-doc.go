package main

import (
	"fmt"
	"insane-doc/logger"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alecthomas/kingpin"
	"gopkg.in/yaml.v2"
)

const (
	termBlockStart = `/*{`
	termBlockEnd   = `}*/`
	termExtractor  = `//!`
	termDecorator  = `//^`
	termItem       = `//*`
	termDesc       = `//>`
	termInsert     = '@'
)

var (
	startTerms = map[string]bool{
		termBlockStart: true,
		termExtractor:  true,
		termDecorator:  true,
		termItem:       true,
	}

	ctx = &context{}

	file = kingpin.Flag("file", "file").Default("./").Short('f').String()

	config = make([]struct {
		Files    []string `yaml:"files"`
		Template string   `yaml:"template"`
	}, 0)

	footer = "\n\n*Generated using [__insane-doc__](https://github.com/vitkovskii/insane-doc)*"
)

type context struct {
	comment    string
	values     map[string]*value
	extractors []*extractor
	decorators []*decorator
}

type dataItem struct {
	comment   string
	key       string
	payload   string
	extracted []string
}
type value struct {
	data []*dataItem
	def  dataItem
}

func getFileLines(filename string) []string {
	content := getFile(filename)

	return strings.Split(content, "\n")
}

func getFile(filename string) string {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.Fatalf(err.Error())
	}
	return string(bytes)
}

func nextLine(lines []string) []string {
	pos := strings.Index(lines[0], termDesc)
	if pos != -1 {
		ctx.comment += strings.TrimSpace(lines[0][pos+len(termDesc):]) + "\n"
	}
	lines = lines[1:]

	return lines
}

func parseOne(lines []string) []string {
	for term := range startTerms {
		pos := strings.Index(lines[0], term)
		// no control structures on this line
		if pos != -1 {
			return parseTerm(lines, term, lines[0][pos+len(term):])
		}
	}

	return nextLine(lines)
}

func addVal(name string, key string, payload string, extracted [] string, comment string) {
	val, has := ctx.values[name]
	if !has {
		val = &value{
			data: make([]*dataItem, 0),
		}
		ctx.values[name] = val
	}

	if comment == "" {
		comment = ctx.comment
		ctx.comment = ""
	}

	d := &val.def
	if key != "" {
		d = &dataItem{}
	}

	d.key = key
	d.comment = comment
	d.payload = payload
	d.extracted = append(make([]string, 0), extracted...)

	if key != "" {
		val.data = append(val.data, d)
	}

	logger.Infof("added val: %s.%s=%s", name, key, payload)
}

func extract(lines [] string) []string {
	values := make([]string, 0)
	for _, extractor := range ctx.extractors {
		values = append(values, extractor.extract(lines[0]))
	}

	return values
}

func parseTerm(lines []string, term string, rest string) []string {
	switch term {
	case termBlockStart:
		parts := strings.Fields(lines[0])
		name := parts[1]
		lines = nextLine(lines)
		text := make([]string, 0, 0)
		for len(lines) > 0 {
			pos := strings.Index(lines[0], termBlockEnd)
			if pos != -1 {
				break
			}

			text = append(text, lines[0])
			lines = nextLine(lines)
		}

		addVal(name, "", strings.Join(text, "\n"), nil, "")

		return lines
	case termExtractor:
		parts := strings.Fields(lines[0])
		parts = parts[1:]
		if len(parts) == 0 {
			logger.Fatalf("empty extractor")
		}
		ctx.extractors = ctx.extractors[:0]
		ctx.decorators = ctx.decorators[:0]
		for _, part := range parts {
			ctx.extractors = append(ctx.extractors, NewExtractor(part))
		}

		if len(ctx.extractors) == 1 {
			ctx.extractors = append(ctx.extractors, NewExtractor(""))
		}

		if len(ctx.extractors) == 2 {
			ctx.extractors = append(ctx.extractors, NewExtractor("undefined"))
		}

		logger.Infof("ctx extractors switched: %s", strings.Join(parts, ", "))
		lines = nextLine(lines)
		return lines
	case termDecorator:
		parts := strings.Fields(lines[0])
		parts = parts[1:]
		if len(parts) == 0 {
			logger.Fatalf("empty decorator")
		}
		ctx.decorators = ctx.decorators[:0]
		for _, part := range parts {
			ctx.decorators = append(ctx.decorators, NewDecorator(part))
		}

		logger.Infof("ctx decorators switched: %s", strings.Join(parts, ", "))
		lines = nextLine(lines)
		return lines
	case termItem:
		extracted := extract(lines)
		for i, decorator := range ctx.decorators {
			extracted[i] = decorator.decorate(extracted[i])
		}
		logger.Infof("item found %s.%s", extracted[0], extracted[1])
		addVal(extracted[0], extracted[1], extracted[2], extracted, rest)
		lines = nextLine(lines)
		return lines
	}

	logger.Panicf("unknown term: %s", term)
	panic("wtf?")
}

func substitute(content string) string {
	result := ""
	for len(content) != 0 {
		st := strings.IndexByte(content, termInsert)
		if st == -1 {
			result += content
			break
		}
		if st == len(content)-1 {
			result += content
			break
		}

		result += content[:st]
		content = content[st+1:]

		command := ""
		for len(content) != 0 {
			c := content[0]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '|' || c == '.' || c == '-' || c == '@' {
				command = command + string(c)
				content = content[1:]
				continue
			}

			break
		}
		// some strange escaping
		if command[0] == termInsert {
			result += command
			continue
		}

		parts := strings.Split(command, "|")
		if len(parts) == 1 {
			parts = append(parts, "plain")
		}

		logger.Infof("command found: %s/%s", parts[1], parts[0])
		cmdResult := doCmd(parts[1], parts[0])
		result += cmdResult
	}

	return result
}

func doCmd(cmd string, valueName string) string {
	value := ctx.values[valueName]
	if value == nil {
		logger.Fatalf("can't find value: %q", valueName)
		panic("_")
	}
	switch cmd {
	case "plain":
		if len(valueName) == 1 && valueName[0] >= '0' && valueName[0] <= '9' {
			return value.def.payload
		}
		return substitute(value.def.payload)
	case "description":
		result := make([]string, 0)
		for _, item := range value.data {
			for i, e := range item.extracted {
				addVal(strconv.Itoa(i+1), "", e, nil, "")
			}

			result = append(result, "### "+item.key)
			result = append(result, "")
			result = append(result, substitute(item.comment))
		}
		return strings.Join(result, "\n")
	case "comment-list":
		result := make([]string, 0)
		for _, item := range value.data {
			for i, e := range item.extracted {
				addVal(strconv.Itoa(i+1), "", e, nil, "")
			}

			result = append(result, "* "+item.comment)
		}
		return strings.Join(result, "\n")
	case "signature-list":
		result := make([]string, 0)
		for _, item := range value.data {
			for i, e := range item.extracted {
				addVal(strconv.Itoa(i+1), "", e, nil, "")
			}

			result = append(result, "`"+item.payload+"`<br>"+item.comment)
		}
		logger.Infof(strings.Join(result, "<br><br>\n"))
		return strings.Join(result, "<br><br>\n")
	case "options":
		result := make([]string, 0)
		for _, data := range value.data {
			result = append(result, data.payload)
		}
		return strings.Join(result, "|")

	case "contents-table":
		result := make([]string, 0)
		for _, data := range value.data {
			result = append(result, fmt.Sprintf("## %s\n%s\n\n[More details...](%s)", data.key, data.comment, data.payload))
		}
		return strings.Join(result, "\n")

	case "links":
		result := make([]string, 0)
		for _, data := range value.data {
			result = append(result, fmt.Sprintf("[%s](%s)", data.key, data.payload))
		}
		return strings.Join(result, ", ")
	}

	logger.Fatalf("unknown command: %q", cmd)
	panic("_")
}

func parseFile(lines []string) {
	for len(lines) > 0 {
		lines = parseOne(lines)
	}
}

func resetCtx() {
	if ctx.values == nil {
		ctx.values = make(map[string]*value)
	}

	for name := range ctx.values {
		// skip global values
		if len(name) > len("global") && name[:6] == "global" {
			continue
		}

		delete(ctx.values, name)
	}

	ctx.decorators = ctx.decorators[:0]
	ctx.extractors = ctx.extractors[:0]
}

func run(files []string, template string) {
	logger.Infof("found template file: %s", template)
	path := filepath.Dir(template)

	out := strings.Replace(template, ".idoc", "", 1)

	resetCtx()

	for _, pattern := range files {
		matches, err := filepath.Glob(filepath.Join(path, pattern))
		if err != nil {
			logger.Fatalf("can't glob files %s: %s", pattern, err.Error())
			panic("_")
		}

		if len(matches) == 0 {
			logger.Infof("no matches found for file pattern %s", pattern)
		}

		for _, file := range matches {
			logger.Infof("adding file %s", file)
			parseFile(getFileLines(file))
		}
	}

	f, err := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {
		logger.Fatalf("can't write output file %s: %s", out, err.Error())
		panic("_")
	}

	content := substitute(getFile(template) + footer)
	_, err = f.WriteString(content)
	if err != nil {
		logger.Fatalf("can't write output file %s: %s", out, err.Error())
	}

	templateDir := filepath.Dir(template)
	contentValName := "global-contents-table-" + strings.ReplaceAll(filepath.Dir(templateDir), "/", "-")
	introduction := ctx.values["introduction"]
	descr := ""
	if introduction != nil {
		descr = introduction.def.payload
	}
	addVal(contentValName, filepath.Base(templateDir), out, nil, descr)

}

func main() {
	kingpin.Parse()

	s, err := os.Stat(*file)
	if err != nil {
		logger.Fatalf(err.Error())
		panic("_")
	}

	insancedocfile := *file
	if s.IsDir() {
		insancedocfile = filepath.Join(insancedocfile, "Insanedocfile")
	}

	cfg := getFile(insancedocfile)
	dir := filepath.Dir(insancedocfile)

	err = os.Chdir(dir)
	if err != nil {
		logger.Fatalf(err.Error())
	}

	err = yaml.Unmarshal([]byte(cfg), &config)
	if err != nil {
		logger.Fatalf(err.Error())
	}

	for _, x := range config {
		matches, err := filepath.Glob(x.Template)
		if err != nil {
			logger.Panicf(err.Error())
		}

		for _, template := range matches {
			run(x.Files, template)
		}
	}
}
