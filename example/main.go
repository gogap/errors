package main

import (
	"fmt"

	"github.com/gogap/errors"
)

var (
	ErrParseTest     = errors.T(10001, "test error")
	ErrParseTest2    = errors.T(10002, "test {{.param1}} error")
	ErrStackTest     = errors.T(10003, "call stack test")
	ErrNamespaceTest = errors.TN("GOOD", 10001, "haha error")
)

func main() {
	if e := errors.LoadMessageTemplate("./test.txt"); e != nil {
		fmt.Println(e)
		return
	}

	err1 := ErrParseTest.New()
	equal1 := ErrParseTest.IsEqual(err1)
	fmt.Println(err1)
	fmt.Println(err1, "Equal", ErrParseTest, "?:", equal1)

	fmt.Println("==FullError=======================")
	fmt.Println(err1.FullError())

	err2 := ErrParseTest2.New(errors.Params{"param1": "example"})

	equal3 := ErrParseTest.IsEqual(err2)
	fmt.Println(ErrParseTest, "Equal", err2, "?:", equal3)

	fmt.Println("==Context=========================")
	fmt.Println(err2.Context())

	fmt.Println("==DeepStackTrace==================")
	errStack := call_1()

	errCode := errStack.(errors.ErrCode)

	fmt.Println(errCode.FullError())

	namedError := ErrNamespaceTest.New()
	fmt.Println(namedError)
	equal4 := ErrParseTest.IsEqual(namedError)
	fmt.Println(ErrParseTest, "Equal", namedError, "?:", equal4)

	e := errors.New("append errors")
	namedError.Append(e)

	fmt.Println(namedError.FullError())

}

func call_1() error {
	return call_2()
}
func call_2() error {
	return call_3()
}
func call_3() error {
	return ErrStackTest.New()
}
