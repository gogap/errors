package errors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"strconv"
	"strings"
	"text/template"

	"github.com/gogap/stack"
	uuid "github.com/nu7hatch/gouuid"
)

const (
	no_VALUE = "<no value>"
)

const (
	ERRCODE_PARSE_TPL_ERROR = 1
	ERRCODE_EXEC_TPL_ERROR  = 2
)

const (
	ERRCODE_NAMESPACE     = "ERRORS"
	DEFAULT_ERR_NAMESPACE = "ERR"
)

var (
	UsingLoadedTemplate = true
)

var (
	errorTemplate  map[string]errCodeTemplate
	errCodeDefined map[string]bool
)

func init() {
	errorTemplate = make(map[string]errCodeTemplate)
	errCodeDefined = make(map[string]bool)
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
	return TN(DEFAULT_ERR_NAMESPACE, code, template)
}

func TN(namespace string, code uint64, template string) errCodeTemplate {
	key := fmt.Sprintf("%s:%d", namespace, code)
	if _, exist := errCodeDefined[key]; exist {
		strErr := fmt.Sprintf("error code %s already exist", key)
		panic(strErr)
	} else {
		errCodeDefined[key] = true
	}
	return errCodeTemplate{code: code, namespace: namespace, template: template}
}

type ErrorContext map[string]interface{}

func (p ErrorContext) String() string {
	if bJson, e := json.Marshal(p); e == nil {
		return string(bJson)
	}
	return ""
}

type ErrCode interface {
	Id() string
	Code() uint64
	Namespace() string
	Error() string
	StackTrace() string
	Context() ErrorContext
	FullError() error
}

type errCodeTemplate struct {
	namespace string
	code      uint64
	template  string
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
		key := fmt.Sprintf("%s:%d", p.namespace, p.code)
		if t, exist := errorTemplate[key]; exist {
			tpl = &t
		}
	}

	strCode := fmt.Sprintf("ERRCODE:%d", tpl.code)

	stack := stack.CallersDeepth(1, 5)

	errId := "<NO-UUID>"
	if errUUID, e := uuid.NewV4(); e == nil {
		errId = errUUID.String()
	}

	crcErrId := crc32.ChecksumIEEE([]byte(errId))

	if t, e := template.New(strCode).Parse(tpl.template); e != nil {
		strErr := fmt.Sprintf("parser error template failed, namespace: %s, code: %d, error: %s", tpl.namespace, tpl.code, e)
		err = &errorCode{id: errId, id_crc: crcErrId, namespace: ERRCODE_NAMESPACE, code: ERRCODE_PARSE_TPL_ERROR, message: strErr, stackTrace: stack.String(), context: params}
		return
	} else {
		var buf bytes.Buffer
		if e := t.Execute(&buf, params); e != nil {
			strErr := fmt.Sprintf("execute template failed, namespace: %s code: %d, error: %s", tpl.namespace, tpl.code, e)
			return &errorCode{id: errId, id_crc: crcErrId, namespace: ERRCODE_NAMESPACE, code: ERRCODE_EXEC_TPL_ERROR, message: strErr, stackTrace: stack.String(), context: params}
		} else {
			bufstr := strings.Replace(buf.String(), no_VALUE, "[NO_VALUE]", -1)
			return &errorCode{id: errId, id_crc: crcErrId, namespace: p.namespace, code: tpl.code, message: bufstr, stackTrace: stack.String(), context: params}
		}
	}
}

func (p *errCodeTemplate) IsEqual(err error) bool {
	if e, ok := err.(ErrCode); ok {
		if e.Code() == p.code && e.Namespace() == p.namespace {
			return true
		}
	}
	return false
}

type errorCode struct {
	id         string
	id_crc     uint32
	code       uint64
	namespace  string
	message    string
	stackTrace string
	context    map[string]interface{}
}

func (p *errorCode) Id() string {
	return p.id
}

func (p *errorCode) Code() uint64 {
	return p.code
}

func (p *errorCode) Namespace() string {
	return p.namespace
}

func (p *errorCode) Error() string {
	return fmt.Sprintf("[%s-%d-%0xd]: %s", p.namespace, p.code, p.id_crc, p.message)
}

func (p *errorCode) FullError() error {
	errLines := make([]string, 1)

	errLines[0] = fmt.Sprintf("ERR_ID: %s\nCODE: %s-%d-%0xd", p.id, p.namespace, p.code, p.id_crc)

	errLines = append(errLines, "MESSAGE:")
	errLines = append(errLines, p.message)
	errLines = append(errLines, "CONTEXT:")
	errLines = append(errLines, p.Context().String())
	errLines = append(errLines, "ORIGINAL_STACK_TRACE:")
	errLines = append(errLines, p.stackTrace)
	return New(strings.Join(errLines, "\n"))
}

func (p *errorCode) Context() ErrorContext {
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

		key := strings.TrimSpace(datas[0])
		tmpl := strings.TrimSpace(datas[1])
		namespace := ""
		strCode := ""

		nameAndCode := strings.Split(key, ":")
		if len(nameAndCode) == 1 {
			strCode = nameAndCode[0]
		} else if len(nameAndCode) == 2 {
			namespace = strings.TrimSpace(nameAndCode[0])
			strCode = nameAndCode[1]
		} else {
			return Errorf("the first column format is NAMESPACE:CODE or CODE, current is: %s, line: %d", key, i+1)
		}

		if namespace == "" {
			namespace = DEFAULT_ERR_NAMESPACE
		}

		if code, e := strconv.ParseUint(strCode, 10, 32); e != nil {
			return e
		} else if code > 0 {
			key := fmt.Sprintf("%s:%d", namespace, code)
			if _, exist := errorTemplate[key]; exist {
				return Errorf("error code of %s already exist, line %d", key, i+1)
			}
			errorTemplate[key] = errCodeTemplate{code: code, namespace: namespace, template: tmpl}
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
