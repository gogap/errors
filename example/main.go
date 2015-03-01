package main

import (
	"fmt"

	"github.com/gogap/errors"
)

var (
	ERR_PARSE_TEST     = errors.T(10001, "test error")
	ERR_PARSE_TEST2    = errors.T(10002, "test {{.param1}} error")
	ERR_STACK_TEST     = errors.T(10003, "call stack test")
	ERR_NAMESPACE_TEST = errors.TN("GOOD", 10001, "haha error")
)

func main() {
	if e := errors.LoadMessageTemplate("./test.txt"); e != nil {
		fmt.Println(e)
		return
	}

	err1 := ERR_PARSE_TEST.New()
	equal1 := ERR_PARSE_TEST.IsEqual(err1)
	fmt.Println(err1)
	fmt.Println(err1, "Equal", ERR_PARSE_TEST, "?:", equal1)

	fmt.Println("==FullError=======================")
	fmt.Println(err1.FullError())

	err2 := ERR_PARSE_TEST2.New(errors.Params{"param1": "example"})

	equal3 := ERR_PARSE_TEST.IsEqual(err2)
	fmt.Println(ERR_PARSE_TEST, "Equal", err2, "?:", equal3)

	fmt.Println("==Context=========================")
	fmt.Println(err2.Context())

	fmt.Println("==DeepStackTrace==================")
	errStack := call_1()

	errCode := errStack.(errors.ErrCode)

	fmt.Println(errCode.FullError())

	namedError := ERR_NAMESPACE_TEST.New()
	fmt.Println(namedError)
	equal4 := ERR_PARSE_TEST.IsEqual(namedError)
	fmt.Println(ERR_PARSE_TEST, "Equal", namedError, "?:", equal4)

	fmt.Println(namedError.FullError())
}

func call_1() error {
	return call_2()
}
func call_2() error {
	return call_3()
}
func call_3() error {
	return ERR_STACK_TEST.New()
}
