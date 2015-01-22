errors
======
For replacement offical package of `errors`

for more detials, see the example

#### `test.txt`
```text
10002 = 测试错误，值为:{{.param1}}
```

#### `main.go`
```go
package main

import (
    "fmt"

    "github.com/gogap/errors"
)

var (
    ERR_PARSE_TEST  = errors.T(10001, "test error")
    ERR_PARSE_TEST2 = errors.T(10002, "test {{.param1}} error")
    ERR_STACK_TEST  = errors.T(10003, "call stack test")
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

    fmt.Println("==StackTrace======================")
    fmt.Println(err1.StackTrace())
    fmt.Println("==Context=========================")
    fmt.Println(err1.Context())
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

    fmt.Println(errCode.StackTrace())
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

```

#### example output
```bash
$ go run main.go

[ERR-10001]: test error
[ERR-10001]: test error Equal {10001 test error} ?: true
==StackTrace======================
github.com/gogap/errors/example/main.go:21 main
/usr/local/go/src/runtime/proc.go:72       main
/usr/local/go/src/runtime/asm_amd64.s:2233 goexit
==Context=========================
{}
==FullError=======================
CODE: 10001
test error

ORIGINAL STACK TRACE:
github.com/gogap/errors/example/main.go:21 main
/usr/local/go/src/runtime/proc.go:72       main
/usr/local/go/src/runtime/asm_amd64.s:2233 goexit
{10001 test error} Equal [ERR-10002]: 测试错误，值为:example ?: false
==Context=========================
{"param1":"example"}
==DeepStackTrace==================
github.com/gogap/errors/example/main.go:56 call_3
github.com/gogap/errors/example/main.go:53 call_2
github.com/gogap/errors/example/main.go:50 call_1
github.com/gogap/errors/example/main.go:42 main
/usr/local/go/src/runtime/proc.go:72       main
/usr/local/go/src/runtime/asm_amd64.s:2233 goexit
```