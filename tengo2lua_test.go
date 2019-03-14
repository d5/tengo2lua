package tengo2lua_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/d5/tengo/assert"
	"github.com/d5/tengo2lua"
	"github.com/yuin/gopher-lua"
)

type ARR = []interface{}
type MAP = map[string]interface{}

func convertEval(t *testing.T, src string, expected interface{}) {
	ls := convert(t, src)
	if !eval(t, ls, expected) {
		t.Logf("Lua Script:\n%s\n", ls)
	}
}

func convert(t *testing.T, src string) string {
	tr := tengo2lua.NewTranspiler([]byte(src), nil)
	out, err := tr.Convert()
	assert.NoError(t, err)
	return out
}

func convertError(t *testing.T, src, expected string) {
	tr := tengo2lua.NewTranspiler([]byte(src), nil)
	ls, err := tr.Convert()
	if !assert.Error(t, err) {
		t.Logf("Lua Script:\n%s\n", ls)
		return
	}
	assert.True(t, strings.Contains(err.Error(), expected), "expected: %s, got: %s", expected, err.Error())
}

func eval(t *testing.T, luaScript string, expected interface{}) bool {
	l := lua.NewState()
	defer l.Close()

	err := l.DoString(luaScript)
	if !assert.NoError(t, err) {
		return false
	}

	return assertEqual(t, expected, fromLV(l.Get(-1)))
}

func assertEqual(t *testing.T, expected, actual interface{}) bool {
	switch expected := expected.(type) {
	case ARR:
		if !assert.Equal(t, len(expected), len(actual.(ARR)), "expected: %v, actual: %v", expected, actual) {
			return false
		}
		for idx, exp := range expected {
			act := actual.(ARR)[idx]
			if !assertEqual(t, exp, act) {
				return false
			}
		}
		return true
	case MAP:
		if !assert.Equal(t, len(expected), len(actual.(MAP)), "expected: %v, actual: %v", expected, actual) {
			return false
		}
		for k, v := range expected {
			act := actual.(MAP)[k]
			if !assertEqual(t, v, act) {
				return false
			}
		}
		return true
	default:
		return assert.Equal(t, expected, actual)
	}
}

func fromLV(v lua.LValue) interface{} {
	switch v := v.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LTable:
		if lua.LVAsBool(v.RawGetString("__a")) {
			return arrayFromLVTable(v)
		}
		return mapFromLVTable(v)
	default:
		panic(fmt.Errorf("unsupported value type: %s (%s)", v.String(), v.Type()))
	}
}

func arrayFromLVTable(v *lua.LTable) ARR {
	var arr ARR
	for i := 0; i < v.Len()+1; i++ {
		arr = append(arr, fromLV(v.RawGet(lua.LNumber(i))))
	}
	return arr
}

func mapFromLVTable(v *lua.LTable) MAP {
	m := make(MAP)
	v.ForEach(func(key lua.LValue, value lua.LValue) {
		m[lua.LVAsString(key)] = fromLV(value)
	})
	return m
}
