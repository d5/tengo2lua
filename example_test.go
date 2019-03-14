package tengo2lua_test

import (
	"fmt"

	"github.com/d5/tengo2lua"
)

func ExampleTranspiler() {
	src := []byte(`
each := func(x, f) { for k, v in x { f(k, v) } }
sum := 0
each([1, 2, 3], func(i, v) { sum += v })
`)

	t := tengo2lua.NewTranspiler(src, nil)
	dst, err := t.Convert()
	if err != nil {
		panic(err)
	}

	fmt.Println(string(dst))
}
