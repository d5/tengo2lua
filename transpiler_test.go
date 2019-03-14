package tengo2lua_test

import "testing"

func TestEval(t *testing.T) {
	convertEval(t, `return 5`, 5.0)
	convertEval(t, `return 2+3`, 5.0)
	convertEval(t, `a:=3; return 2+a`, 5.0)
	convertEval(t, `a:=3; a=2; return 2+a`, 4.0)
	convertEval(t, `a:=3; b:=a+2; return a+b`, 8.0)

	// variable init and scopes
	convertError(t, `a=5`, "unresolved reference 'a'")
	convertError(t, `a:=5; a:=2`, "'a' redeclared")
	convertError(t, `if a:=3;a>0 {}; a=10`, "unresolved reference 'a'")
	convertError(t, `for a:=0;a<5;a++ {}; a=10`, "unresolved reference 'a'")
	convertError(t, `if a:=3;a>0 {}; return a`, "unresolved reference 'a'")
	convertError(t, `for a:=0;a<5;a++ {}; return a`, "unresolved reference 'a'")

	// if-else statement
	convertEval(t, `a:=3; if a>2 { a=5 }; return a `, 5.0)
	convertEval(t, `a:=1; if a>2 { a=5 }; return a `, 1.0)
	convertEval(t, `a:=3; if a>2 { return "foo" } else { return "bar" }`, "foo")
	convertEval(t, `a:=1; if a>2 { return "foo" } else { return "bar" }`, "bar")
	convertEval(t, `a:=1; if a==1 { return "one" } else if a==2 { return "two" } else { return "foo" }`, "one")
	convertEval(t, `a:=2; if a==1 { return "one" } else if a==2 { return "two" } else { return "foo" }`, "two")
	convertEval(t, `a:=0; if a==1 { return "one" } else if a==2 { return "two" } else { return "foo" }`, "foo")

	// array and map
	convertEval(t, `return [1,2,3]`, ARR{1.0, 2.0, 3.0})
	convertEval(t, `return [1,"two",false,[4,5,6]]`, ARR{1.0, "two", false, ARR{4.0, 5.0, 6.0}})
	convertEval(t, `return {a:1,b:2,c:3}`, MAP{"a": 1.0, "b": 2.0, "c": 3.0})
	convertEval(t, `return {a:1,b:"two",c:{d:false,e:"foo",f:9}}`, MAP{"a": 1.0, "b": "two", "c": MAP{"d": false, "e": "foo", "f": 9.0}})
	convertEval(t, `return {a:1,b:"two",c:[4,5,6]}`, MAP{"a": 1.0, "b": "two", "c": ARR{4.0, 5.0, 6.0}})

	// array indexing
	convertEval(t, `return [1,2,3][-1]`, nil)
	convertEval(t, `return [1,2,3][0]`, 1.0)
	convertEval(t, `return [1,2,3][1]`, 2.0)
	convertEval(t, `return [1,2,3][2]`, 3.0)
	convertEval(t, `return [1,2,3][3]`, nil)

	// slicing
	//convertEval(t, `return [1,2,3][0:3]`, ARR{1.0, 2.0, 3.0})
	//convertEval(t, `return [1,2,3][1:2]`, ARR{2.0})
	convertEval(t, `return "012345"[0:2]`, "01")
	convertEval(t, `return "012345"[1:5]`, "1234")
	convertEval(t, `return "012345"[4:6]`, "45")

	// for statement
	convertEval(t, `s:=0; for i:=1;i<=5;i++ { s+=i }; return s`, 15.0)
	convertEval(t, `i:=0; for i<5 { i++ }; return i`, 5.0)
	convertEval(t, `i:=0; for { i++; if i==3 { return i } }; return i`, 3.0)
	convertEval(t, `i:=0; for ;i<5;i++ { if i==3 { break } }; return i`, 3.0)
	convertEval(t, `i:=0; for ;i<5;i++ { if i==3 { continue } }; return i`, 5.0)
	convertEval(t, `a:=0; for i:=0;i<5;i++ { if i==3 { break }; a=i }; return a`, 2.0)
	convertEval(t, `a:=0; for i:=0;i<5;i++ { a=i; if i==3 { break } }; return a`, 3.0)
	convertEval(t, `a:=0; for i:=0;i<5;i++ { if i==3 { continue }; a+=i }; return a`, 7.0)
	convertEval(t, `a:=0; for i:=0;i<5;i++ { a+=i; if i==3 { continue } }; return a`, 10.0)
	// nested for loops
	convertEval(t, `s:=0; for i:=1;i<=2;i++ { for j:=1;j<=3;j++ { s+=i*j } }; return s`, 18.0)                       // 1+2+3+2+4+6
	convertEval(t, `s:=0; for i:=1;i<=2;i++ { for j:=1;j<=3;j++ { if j==2 { break }; s+=i*j } }; return s`, 3.0)     // 1+2
	convertEval(t, `s:=0; for i:=1;i<=2;i++ { for j:=1;j<=3;j++ { if j==2 { continue }; s+=i*j } }; return s`, 12.0) // 1+3+2+6
	convertEval(t, `s:=0; for i:=1;i<=3;i++ { if i==2 { break }; for j:=1;j<=2;j++ { s+=i*j } }; return s`, 3.0)     // 1+2
	convertEval(t, `s:=0; for i:=1;i<=3;i++ { if i==2 { continue }; for j:=1;j<=2;j++ { s+=i*j } }; return s`, 12.0) // 1+3+2+6

	// for-in statement
	convertEval(t, `s:=0; a:=[2,4,6]; for i, v in a { s+=i }; return s`, 3.0)
	convertEval(t, `s:=0; a:=[2,4,6]; for i, _ in a { s+=i }; return s`, 3.0)
	convertEval(t, `s:=0; a:=[2,4,6]; for v in a { s+=v }; return s`, 12.0)
	convertEval(t, `s:=0; a:=[2,4,6]; for _, v in a { s+=v }; return s`, 12.0)
	convertEval(t, `s:=0; a:=[2,4,6]; for i, v in a { s+=v }; return s`, 12.0)
	convertEval(t, `s:=0; a:=[2,4,6]; for i, v in a { s+=i+v }; return s`, 15.0)
	convertEval(t, `s:=""; a:={a:2,b:4,c:6}; for k, v in a { s+=k }; return s`, "abc")
	convertEval(t, `s:=0; a:={a:2,b:4,c:6}; for k, v in a { s+=v }; return s`, 12.0)
	convertEval(t, `s:=0; a:={a:2,b:4,c:6}; for v in a { s+=v }; return s`, 12.0)
	convertEval(t, `s:=0; a:={a:2,b:4,c:6}; for _, v in a { s+=v }; return s`, 12.0)
	convertEval(t, `s:=""; a:={a:2,b:4,c:6}; for k, _ in a { s+=k }; return s`, "abc")
	convertEval(t, `s:=0; a:=[2,4,6]; for i, v in a { if i==1 { break }; s+=v }; return s`, 2.0)
	convertEval(t, `s:=0; a:=[2,4,6]; for i, v in a { if i==1 { continue }; s+=v }; return s`, 8.0)

	// string concatenation
	convertEval(t, `return "foo" + "bar"`, "foobar")

	// len builtin
	convertEval(t, `return len([])`, 0.0)
	convertEval(t, `return len([1])`, 1.0)
	convertEval(t, `return len([1,5,10])`, 3.0)
	convertEval(t, `return len({})`, 0.0)
	convertEval(t, `return len({a:1})`, 1.0)
	convertEval(t, `return len({a:1,b:5,c:10})`, 3.0)
	convertEval(t, `return len("")`, 0.0)
	convertEval(t, `return len("123")`, 3.0)

	// function and function calls
	convertEval(t, `a:=func(){return 5}; return a()`, 5.0)
	convertEval(t, `a:=func(x,y){return x+y}; return a(1,2)`, 3.0)
	convertEval(t, `a:=func(x,y){return x+y}; return a(1+2,2+4)`, 9.0)
	convertEval(t, `c:=1; a:=func(x,y){return x+y}; return a(c+2,c+3)`, 7.0)
	convertEval(t, `c:=1; a:=func(x,y){xy:=x+y; return xy}; return a(c+2,c+3)`, 7.0)

	// conditional expression
	convertEval(t, `return 5>3?"foo":"bar"`, "foo")
}
