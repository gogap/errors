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
)

func main() {
    if e := errors.LoadMessageTemplate("./test.txt"); e != nil {
        fmt.Println(e)
        return
    }

    err1 := ERR_PARSE_TEST.New()
    equal1 := ERR_PARSE_TEST.IsEqual(err1)
    fmt.Println(err1)
    fmt.Println(equal1)
    fmt.Println(err1.StackTrace())
    fmt.Println(err1.Context())
    fmt.Println(err1.FullError())

    err2 := ERR_PARSE_TEST2.New(errors.Params{"param1": "example"})
    equal2 := ERR_PARSE_TEST2.IsEqual(err2)
    fmt.Println(err2)
    fmt.Println(equal2)

    equal3 := ERR_PARSE_TEST.IsEqual(err2)
    fmt.Println(equal3)
}

```