package errors

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"text/template"
)

const (
	no_VALUE = "<no value>"
)

const (
	ERRCODE_PARSE_TPL_ERROR = 100
	ERRCODE_EXEC_TPL_ERROR  = 101
)

var (
	UsingLoadedTemplate = true
)

var (
	errorTemplate  map[uint64]errCodeTemplate
	errCodeDefined map[uint64]bool
)

func init() {
	errorTemplate = make(map[uint64]errCodeTemplate)
	errCodeDefined = make(map[uint64]bool)
}

type Params map[string]interface{}

type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

func New(text string) error {
	return &errorString{text}
}

func Errorf(format string, a ...interface{}) error {
	return fmt.Errorf(format, a...)
}

func T(code uint64, template string) errCodeTemplate {
	if _, exist := errCodeDefined[code]; exist {
		strErr := fmt.Sprintf("error code %d already exist", code)
		panic(strErr)
	} else {
		errCodeDefined[code] = true
	}
	return errCodeTemplate{code: code, template: template}
}

type ErrCode interface {
	Code() uint64
	Error() string
	StackTrace() string
	Context() string
	FullError() error
}

type errCodeTemplate struct {
	code     uint64
	template string
}

func (p *errCodeTemplate) New(v ...Params) (err ErrCode) {
	params := Params{}
	if v != nil {
		for _, param := range v {
			for pn, pv := range param {
				params[pn] = pv
			}
		}
	}

	var tpl *errCodeTemplate = p
	if UsingLoadedTemplate {
		if t, exist := errorTemplate[p.code]; exist {
			tpl = &t
		}
	}

	strCode := fmt.Sprintf("ERRCODE:%d", tpl.code)
	st, ctx := StackTrace()
	if t, e := template.New(strCode).Parse(tpl.template); e != nil {
		strErr := fmt.Sprintf("parser error template failed, code: %d, error: %s", tpl.code, e)
		err = &errorCode{code: ERRCODE_PARSE_TPL_ERROR, message: strErr, stackTrace: st, context: ctx}
		return
	} else {
		var buf bytes.Buffer
		if e := t.Execute(&buf, params); e != nil {
			strErr := fmt.Sprintf("execute template failed, code: %d, error: %s", tpl.code, e)
			return &errorCode{code: ERRCODE_EXEC_TPL_ERROR, message: strErr, stackTrace: st, context: ctx}
		} else {
			bufstr := strings.Replace(buf.String(), no_VALUE, "[NO_VALUE]", -1)
			return &errorCode{code: tpl.code, message: bufstr, stackTrace: st, context: ctx}
		}
	}
}

func (p *errCodeTemplate) IsEqual(err error) bool {
	if e, ok := err.(ErrCode); ok {
		if e.Code() == p.code {
			return true
		}
	}
	return false
}

type errorCode struct {
	code       uint64
	message    string
	stackTrace string
	context    string
}

func (p *errorCode) Code() uint64 {
	return p.code
}

func (p *errorCode) Error() string {
	return fmt.Sprintf("[ERR-%d]: %s", p.code, p.message)
}

func (p *errorCode) FullError() error {
	errLines := make([]string, 1)
	errLines[0] = fmt.Sprintf("CODE: %d", p.code)
	errLines = append(errLines, p.message)
	errLines = append(errLines, "")
	errLines = append(errLines, "ORIGINAL STACK TRACE:")
	errLines = append(errLines, p.stackTrace)
	return New(strings.Join(errLines, "\n"))
}

func (p *errorCode) Context() string {
	return p.context
}

func (p *errorCode) StackTrace() string {
	return p.stackTrace
}

func LoadMessageTemplate(fileName string) error {
	var fileLines []string
	if bFile, e := ioutil.ReadFile(fileName); e != nil {
		return e
	} else {
		fileLines = strings.Split(string(bFile), "\n")
	}

	if len(fileLines) == 0 {
		return nil
	}

	for i, line := range fileLines {

		line = strings.TrimSpace(line)
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}

		datas := strings.Split(line, "=")

		if len(datas) != 2 {
			return Errorf("the error code template column count is not equal 2, line: %d", i+1)
		}

		strCode := strings.TrimSpace(datas[0])
		tmpl := strings.TrimSpace(datas[1])

		if code, e := strconv.ParseUint(strCode, 10, 32); e != nil {
			return e
		} else if code > 0 {
			if _, exist := errorTemplate[code]; exist {
				return Errorf("error code of %d already exist, line %d", code, i+1)
			}
			errorTemplate[code] = errCodeTemplate{code: code, template: tmpl}
		} else {
			return Errorf("error code should greater than 0, line %d", i+1)
		}
	}
	return nil
}

func IsErrCode(err error) bool {
	_, ok := err.(ErrCode)
	return ok
}

func StackTrace() (current, context string) {
	return stackTrace(3)
}

func stackTrace(skip int) (current, context string) {
	buf := make([]byte, 128)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, len(buf)*2)
	}

	indexNewline := func(b []byte, start int) int {
		if start >= len(b) {
			return len(b)
		}
		searchBuf := b[start:]
		index := bytes.IndexByte(searchBuf, '\n')
		if index == -1 {
			return len(b)
		} else {
			return (start + index)
		}
	}

	var strippedBuf bytes.Buffer
	index := indexNewline(buf, 0)
	if index != -1 {
		strippedBuf.Write(buf[:index])
	}

	for i := 0; i < skip; i++ {
		index = indexNewline(buf, index+1)
		index = indexNewline(buf, index+1)
	}

	isDone := false
	startIndex := index
	lastIndex := index
	for !isDone {
		index = indexNewline(buf, index+1)
		if (index - lastIndex) <= 1 {
			isDone = true
		} else {
			lastIndex = index
		}
	}
	strippedBuf.Write(buf[startIndex:index])
	return strippedBuf.String(), string(buf[index:])
}
