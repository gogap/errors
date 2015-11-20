package errors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rs/xid"
	"hash/crc32"
	"io/ioutil"
	"strconv"
	"strings"
	"text/template"

	"github.com/gogap/stack"
)

const (
	no_VALUE = "<no value>"
)

const (
	ErrcodeParseTmplError = 1
	ErrcodeExecTmpleError = 2
)

const (
	ErrcodeNamespace      = "ERRCODE"
	DefaultErrorNamespace = "ERR"
)

var (
	UsingLoadedTemplate = true
)

var (
	errorTemplate  map[string]*ErrCodeTemplate
	errCodeDefined map[string]bool
)

func init() {
	errorTemplate = make(map[string]*ErrCodeTemplate)
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

func T(code uint64, template string) ErrCodeTemplate {
	return TN(DefaultErrorNamespace, code, template)
}

func TN(namespace string, code uint64, template string) ErrCodeTemplate {
	key := fmt.Sprintf("%s:%d", namespace, code)
	if _, exist := errCodeDefined[key]; exist {
		strErr := fmt.Sprintf("error code %s already exist", key)
		panic(strErr)
	} else {
		errCodeDefined[key] = true
	}
	return ErrCodeTemplate{code: code, namespace: namespace, namespaceAlias: namespace, template: template}
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
	Append(err ...interface{}) ErrCode
}

type ErrCodeTemplate struct {
	namespace      string
	namespaceAlias string
	code           uint64
	template       string
}

func (p *ErrCodeTemplate) New(v ...Params) (err ErrCode) {
	params := Params{}
	if v != nil {
		for _, param := range v {
			for pn, pv := range param {
				params[pn] = pv
			}
		}
	}

	var tpl *ErrCodeTemplate = p
	if UsingLoadedTemplate {
		key := fmt.Sprintf("%s:%d", p.namespace, p.code)
		if t, exist := errorTemplate[key]; exist {
			tpl = t
		}
	}

	strCode := fmt.Sprintf("ErrCode:%d", tpl.code)

	stack := stack.CallersDeepth(1, 5)

	errId := xid.New().String()

	crcErrId := crc32.ChecksumIEEE([]byte(errId))
	strCRCErrId := fmt.Sprintf("%0X", crcErrId)
	if len(strCRCErrId) > 7 {
		strCRCErrId = strCRCErrId[0:7]
	}

	if t, e := template.New(strCode).Parse(tpl.template); e != nil {
		strErr := fmt.Sprintf("parser error template failed, namespace: %s, code: %d, error: %s", tpl.namespaceAlias, tpl.code, e)
		err = &errorCode{id: strCRCErrId, namespace: ErrcodeNamespace, code: ErrcodeParseTmplError, message: strErr, stackTrace: stack.String(), context: params}
		return
	} else {
		var buf bytes.Buffer
		if e := t.Execute(&buf, params); e != nil {
			strErr := fmt.Sprintf("execute template failed, namespace: %s code: %d, error: %s", tpl.namespaceAlias, tpl.code, e)
			return &errorCode{id: strCRCErrId, namespace: ErrcodeNamespace, code: ErrcodeExecTmpleError, message: strErr, stackTrace: stack.String(), context: params}
		} else {
			bufstr := strings.Replace(buf.String(), no_VALUE, "[NO_VALUE]", -1)
			return &errorCode{id: strCRCErrId, namespace: tpl.namespaceAlias, code: tpl.code, message: bufstr, stackTrace: stack.String(), context: params}
		}
	}
}

func (p *ErrCodeTemplate) IsEqual(err error) bool {
	if e, ok := err.(ErrCode); ok {
		if e.Code() == p.code && e.Namespace() == p.namespace {
			return true
		}
	}
	return false
}

type errorCode struct {
	id         string
	code       uint64
	namespace  string
	message    string
	stackTrace string
	context    map[string]interface{}
	errors     []string
}

func NewErrorCode(id string, code uint64, namespace string, message string, stackTrace string, context map[string]interface{}) ErrCode {
	return &errorCode{
		id:         id,
		code:       code,
		namespace:  namespace,
		message:    message,
		stackTrace: stackTrace,
		context:    context,
	}
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
	msg := p.message
	if p.errors != nil && len(p.errors) > 0 {
		if strings.TrimSpace(p.message) != "" {
			msg = msg + ", error: "
		} else {
			msg = msg + "error: "
		}

		msg = msg + strings.Join(p.errors, "; ")
		msg = msg + "."
	}

	return msg
}

func (p *errorCode) FullError() error {
	errLines := make([]string, 1)

	errLines[0] = fmt.Sprintf("Id: %s#%d:%s", p.namespace, p.code, p.id)

	errLines = append(errLines, "Error:")
	errLines = append(errLines, p.Error())
	errLines = append(errLines, "Context:")
	errLines = append(errLines, p.Context().String())
	errLines = append(errLines, "StackTrace:")
	errLines = append(errLines, p.stackTrace)
	return New(strings.Join(errLines, "\n"))
}

func (p *errorCode) Context() ErrorContext {
	return p.context
}

func (p *errorCode) StackTrace() string {
	return p.stackTrace
}

func (p *errorCode) Append(err ...interface{}) ErrCode {
	if err != nil {
		for _, e := range err {
			p.errors = append(p.errors, fmt.Sprintf("%v", e))
		}
	}
	return p
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
		namespaceAlias := ""
		strCode := ""

		nameAndCode := strings.Split(key, "|")
		if len(nameAndCode) == 1 {
			strCode = strings.TrimSpace(nameAndCode[0])
		} else if len(nameAndCode) == 2 {
			namespace = strings.TrimSpace(nameAndCode[0])
			strCode = strings.TrimSpace(nameAndCode[1])
		} else {
			return Errorf("the first column format is NAMESPACE|CODE or CODE, current is: %s, line: %d", key, i+1)
		}

		nameAliasAndTmpl := strings.Split(tmpl, "|")
		if len(nameAliasAndTmpl) == 1 {
			tmpl = strings.TrimSpace(nameAliasAndTmpl[0])
		} else if len(nameAliasAndTmpl) == 2 {
			namespaceAlias = strings.TrimSpace(nameAliasAndTmpl[0])
			tmpl = strings.TrimSpace(nameAliasAndTmpl[1])
		} else {
			return Errorf("the second column format is NAMESPACE_ALAIS|TEMPLATE or CODE, current is: %s, line: %d", tmpl, i+1)
		}

		if namespace == "" {
			namespace = DefaultErrorNamespace
		}

		if namespaceAlias == "" {
			namespaceAlias = namespace
		}

		if code, e := strconv.ParseUint(strCode, 10, 32); e != nil {
			return e
		} else if code > 0 {
			key := fmt.Sprintf("%s:%d", namespace, code)
			if _, exist := errorTemplate[key]; exist {
				return Errorf("error code of %s already exist, line %d", key, i+1)
			}
			errorTemplate[key] = &ErrCodeTemplate{code: code, namespace: namespace, namespaceAlias: namespaceAlias, template: tmpl}
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
