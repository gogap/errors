package errors

import (
	"bytes"
	"fmt"
	"io/ioutil"
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
	if code < 10000 {
		panic("error code should greater than 10000")
	}

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
	if t, e := template.New(strCode).Parse(tpl.template); e != nil {
		strErr := fmt.Sprintf("parser error template failed, code: %d, error: %s", tpl.code, e)
		err = &errorCode{code: ERRCODE_PARSE_TPL_ERROR, message: strErr}
		return
	} else {
		var buf bytes.Buffer
		if e := t.Execute(&buf, params); e != nil {
			strErr := fmt.Sprintf("execute template failed, code: %d, error: %s", tpl.code, e)
			return &errorCode{code: ERRCODE_EXEC_TPL_ERROR, message: strErr}
		} else {
			bufstr := strings.Replace(buf.String(), no_VALUE, "[NO_VALUE]", -1)
			return &errorCode{code: tpl.code, message: bufstr}
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
	code    uint64
	message string
}

func (p *errorCode) Code() uint64 {
	return p.code
}

func (p *errorCode) Error() string {
	return p.message
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
