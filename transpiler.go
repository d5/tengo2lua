package tengo2lua

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/d5/tengo/compiler"
	"github.com/d5/tengo/compiler/ast"
	"github.com/d5/tengo/compiler/parser"
	"github.com/d5/tengo/compiler/source"
	"github.com/d5/tengo/compiler/token"
)

var (
	reservedVarName = regexp.MustCompile(`^__.+__$`)
)

// Transpiler converts Tengo source code into Lua code.
type Transpiler struct {
	src              []byte
	file             *source.File
	symbolTable      *compiler.SymbolTable
	loopDepth        int
	indentLevel      int
	options          *Options
	builtinFuncsUsed map[string]bool
	helpersUsed      map[helper]bool
}

// NewTranspiler creates a new Transpiler.
func NewTranspiler(src []byte, opts *Options) *Transpiler {
	fileSet := source.NewFileSet()
	file := fileSet.AddFile("script", -1, len(src))

	if opts == nil {
		opts = DefaultOptions()
	}

	return &Transpiler{
		src:              src,
		file:             file,
		symbolTable:      compiler.NewSymbolTable(),
		builtinFuncsUsed: make(map[string]bool),
		helpersUsed:      make(map[helper]bool),
		options:          opts,
	}
}

// Convert converts the input Tengo source code.
func (t *Transpiler) Convert() (output string, err error) {
	astFile, err := t.parse()
	if err != nil {
		return
	}

	t.indentLevel = 0

	output, err = t.convert(astFile)
	if err != nil {
		return
	}

	output = t.helperCode() + output

	// TODO: add option to minify the output code
	// e.g. remove all redundant whitespaces
	// ...

	return
}

// TODO: helper functions should be added selectively
// e.g. add __iter__ only when the code actually had "for-in" statement.
func (t *Transpiler) helperCode() string {
	var out string

	for h, used := range t.helpersUsed {
		if used {
			out += helpers[h] + "\n"
		}
	}

	for name, used := range t.builtinFuncsUsed {
		if used {
			out += name + " = " + builtinFunctions[name] + "\n"
		}
	}

	return out
}

func (t *Transpiler) convert(node ast.Node) (string, error) {
	switch node := node.(type) {
	case *ast.File:
		lines := ""
		for _, stmt := range node.Stmts {
			out, err := t.convert(stmt)
			if err != nil {
				return "", err
			}
			lines += out
		}
		return lines + "\n", nil

	case *ast.BlockStmt:
		lines := ""
		for _, stmt := range node.Stmts {
			out, err := t.convert(stmt)
			if err != nil {
				return "", err
			}
			lines += out
		}
		return lines, nil

	case *ast.ExprStmt:
		out, err := t.convert(node.Expr)
		if err != nil {
			return "", err
		}

		return t.line(out), nil

	case *ast.IncDecStmt:
		// expand to "expr = expr + 1"
		expr, err := t.convert(node.Expr)
		if err != nil {
			return "", err
		}

		op := "+"
		if node.Token == token.Dec {
			op = "-"
		}

		return t.line(expr + "=" + expr + op + "1"), nil

	case *ast.AssignStmt:
		out, err := t.convertAssignment(node, node.LHS, node.RHS, node.Token)
		if err != nil {
			return "", err
		}
		return t.line(out), nil

	case *ast.ParenExpr:
		expr, err := t.convert(node.Expr)
		if err != nil {
			return "", err
		}

		return "(" + expr + ")", nil

	case *ast.BinaryExpr:
		left, err := t.convert(node.LHS)
		if err != nil {
			return "", err
		}
		right, err := t.convert(node.RHS)
		if err != nil {
			return "", err
		}

		var op string
		switch node.Token {
		case token.LAnd:
			op = "and"
		case token.LOr:
			op = "or"
		case token.NotEqual:
			op = "~="
		case token.And, token.Or, token.Xor, token.AndNot, token.Shl, token.Shr:
			// TODO: bitwise operators not supported
			return "", t.error(node, "operator "+node.Token.String()+"not supported")
		default:
			op = node.Token.String()
		}

		// potentially string + operator was used
		if node.Token == token.Add {
			t.helpersUsed[helperStringConcat] = true
		}

		return "(" + left + " " + op + " " + right + ")", nil

	case *ast.IntLit:
		return node.Literal, nil // as number

	case *ast.FloatLit:
		return node.Literal, nil // as number

	case *ast.BoolLit:
		if node.Value {
			return "true", nil
		} else {
			return "false", nil
		}

	case *ast.StringLit:
		return strconv.Quote(node.Value), nil

	case *ast.CharLit:
		// TODO: character literal not supported
		return "", t.error(node, "character literal not supported")

	case *ast.UndefinedLit:
		return "nil", nil

	case *ast.UnaryExpr:
		expr, err := t.convert(node.Expr)
		if err != nil {
			return "", err
		}

		switch node.Token {
		case token.Not:
			return "(not (" + expr + "))", nil
		case token.Sub:
			return "(-(" + expr + "))", nil
		case token.Xor:
			return "", t.error(node, "binary complement not supported")
		case token.Add:
			return "(+(" + expr + "))", nil // TODO: is this even valid?
		default:
			return "", t.error(node, "invalid unary operator: %s", node.Token.String())
		}

	case *ast.ReturnStmt:
		if node.Result == nil {
			return t.line("return"), nil
		} else {
			expr, err := t.convert(node.Result)
			if err != nil {
				return "", err
			}

			return t.line("return (%s)", expr), nil
		}

	case *ast.SelectorExpr:
		expr, err := t.convert(node.Expr)
		if err != nil {
			return "", err
		}
		index, err := t.convert(node.Sel)
		if err != nil {
			return "", err
		}
		return "(" + expr + ")[" + index + "]", nil

	case *ast.IndexExpr:
		expr, err := t.convert(node.Expr)
		if err != nil {
			return "", err
		}
		index, err := t.convert(node.Index)
		if err != nil {
			return "", err
		}
		return "(" + expr + ")[" + index + "]", nil

	case *ast.Ident:
		_, _, ok := t.symbolTable.Resolve(node.Name)
		if !ok {
			// check builtin function name
			if _, ok := builtinFunctions[node.Name]; ok {
				t.builtinFuncsUsed[node.Name] = true
				return node.Name, nil
			}

			return "", t.error(node, "unresolved reference '%s'", node.Name)
		}

		return node.Name, nil

	case *ast.IfStmt:
		// open new symbol table for the statement
		t.symbolTable = t.symbolTable.Fork(true)
		defer func() {
			t.symbolTable = t.symbolTable.Parent(false)
		}()

		// do
		//   (init statement)
		//   if (condition expression) then
		//      (body)
		//   else
		//      (else body)
		//   end
		// end

		out := t.line("do")
		t.indentLevel++

		if node.Init != nil {
			init, err := t.convert(node.Init)
			if err != nil {
				return "", err
			}
			out += t.line(init)
		}

		cond, err := t.convert(node.Cond)
		if err != nil {
			return "", err
		}
		out += t.line("if (%s) then", cond)
		t.indentLevel++

		body, err := t.convert(node.Body)
		if err != nil {
			return "", err
		}
		out += body
		t.indentLevel--

		if node.Else != nil {
			out += t.line("else")
			t.indentLevel++

			elseBody, err := t.convert(node.Else)
			if err != nil {
				return "", err
			}
			out += elseBody
			t.indentLevel--
		}

		out += t.line("end")

		t.indentLevel--
		out += t.line("end")

		return out, nil

	case *ast.ForStmt:
		// open new symbol table for the statement
		t.symbolTable = t.symbolTable.Fork(true)
		defer func() {
			t.symbolTable = t.symbolTable.Parent(false)
		}()

		t.loopDepth++
		defer func() { t.loopDepth-- }()

		// do
		//   (init statement)
		//   while (condition expression) do
		//     local __cont__ = false
		//     repeat
		//       (body)
		//       __cont__ = true
		//	   until 1
		//     if __cont__ then
		//       (post statement)
		//     else
		//     	 break
		//     end
		//   end
		// end
		//
		//  inside (body)
		//    - Tengo "break" will simply break from the inner loop
		//    - Tengo "continue " will set '__cont__' to true, then break from the inner loop

		var out string

		// init
		if node.Init != nil {
			out += t.line("do")
			t.indentLevel++

			init, err := t.convert(node.Init)
			if err != nil {
				return "", err
			}

			out += init
		}

		// while (cond) do
		if node.Cond != nil {
			cond, err := t.convert(node.Cond)
			if err != nil {
				return "", err
			}
			out += t.line("while (%s) do", cond)
		} else {
			out += t.line("while true do")
		}
		t.indentLevel++

		// local __cont__ = false
		var contVarName = t.continueVarName()
		out += t.line("local %s = false", contVarName)

		// repeat
		out += t.line("repeat")
		t.indentLevel++

		// (body)
		body, err := t.convert(node.Body)
		if err != nil {
			return "", err
		}
		out += body

		// __cont__ = true
		out += t.line("%s = true", contVarName)

		// until 1
		t.indentLevel--
		out += t.line("until 1")

		// if __cont__ then
		out += t.line("if %s then", contVarName)
		t.indentLevel++

		// (post)
		if node.Post != nil {
			post, err := t.convert(node.Post)
			if err != nil {
				return "", err
			}
			out += post
		}

		// else
		t.indentLevel--
		out += t.line("else")
		t.indentLevel++

		// break
		out += t.line("break")

		// end
		t.indentLevel--
		out += t.line("end")

		// end
		t.indentLevel--
		out += t.line("end")

		if node.Init != nil {
			// end
			t.indentLevel--
			out += t.line("end")
		}

		return out, nil

	case *ast.ForInStmt:
		// open new symbol table for the statement
		t.symbolTable = t.symbolTable.Fork(true)
		defer func() {
			t.symbolTable = t.symbolTable.Parent(false)
		}()

		t.loopDepth++
		defer func() { t.loopDepth-- }()

		// for (key), (value) in __iter__(seq) do
		//   local __cont__ = false
		//   repeat
		//     (body)
		//     __cont__ = true
		//	 until 1
		//   if not __cont__ then break end
		// end
		//
		//  inside (body)
		//    - Tengo "break" will simply break from the inner loop
		//    - Tengo "continue " will set '__cont__' to true, then break from the inner loop

		// for (key), (value) in pairs(seq) do
		keyVarName := node.Key.Name
		if keyVarName != "_" {
			t.symbolTable.Define(keyVarName)
		}
		valueVarName := node.Value.Name
		if valueVarName != "_" {
			t.symbolTable.Define(valueVarName)
		}
		iterable, err := t.convert(node.Iterable)
		if err != nil {
			return "", err
		}
		out := t.line("for %s, %s in __iter__(%s) do", keyVarName, valueVarName, iterable)
		t.indentLevel++

		// local __cont__ = false
		var contVarName = t.continueVarName()
		out += t.line("local %s = false", contVarName)

		// repeat
		out += t.line("repeat")
		t.indentLevel++

		// (body)
		body, err := t.convert(node.Body)
		if err != nil {
			return "", err
		}
		out += body

		// __cont__ = true
		out += t.line("%s = true", contVarName)

		// until 1
		t.indentLevel--
		out += t.line("until 1")

		// if not __cont__ then break end
		out += t.line("if not %s then break end", contVarName)

		// end
		t.indentLevel--
		out += t.line("end")

		t.helpersUsed[helperIterator] = true

		return out, nil

	case *ast.BranchStmt:
		if node.Token == token.Break {
			return t.line("break"), nil
		} else if node.Token == token.Continue {
			return t.line(t.continueVarName()+" = true") + t.line("break"), nil
		} else {
			panic(fmt.Errorf("invalid branch statement: %s", node.Token.String()))
		}

	case *ast.ArrayLit:
		// Index1 == false
		//   { [0]=nil, arr=true }
		//   { [0]=elem1, elem2, elem3, arr=true }

		if len(node.Elements) == 0 {
			return "{[0]=nil, __a=true}", nil
		}

		var out []string
		for _, elem := range node.Elements {
			expr, err := t.convert(elem)
			if err != nil {
				return "", err
			}
			out = append(out, "("+expr+")")
		}
		return "{[0]=" + strings.Join(out, ",") + ", __a=true}", nil

	case *ast.MapLit:
		// { ["key1"] = value1, ["key2"] = value2 }

		var out []string
		for _, elt := range node.Elements {
			val, err := t.convert(elt.Value)
			if err != nil {
				return "", err
			}

			out = append(out, "["+strconv.Quote(elt.Key)+"]=("+val+")")
		}

		return "{" + strings.Join(out, ",") + "}", nil

	case *ast.SliceExpr:
		// {unpack(expr, low, high-1)}

		expr, err := t.convert(node.Expr)
		if err != nil {
			return "", err
		}

		low, err := t.convert(node.Low)
		if err != nil {
			return "", err
		}

		high, err := t.convert(node.High)
		if err != nil {
			return "", err
		}

		t.helpersUsed[helperSlicing] = true

		return "__slice__(" + expr + "," + low + "," + high + ")", nil

	case *ast.CallExpr:
		ident, err := t.convert(node.Func)
		if err != nil {
			return "", err
		}

		var args []string
		for _, a := range node.Args {
			arg, err := t.convert(a)
			if err != nil {
				return "", err
			}
			args = append(args, arg)
		}

		//if fn, ok := ConvertFunctions[ident]; ok {
		//	return fn(args)
		//}

		return ident + "(" + strings.Join(args, ",") + ")", nil

	case *ast.FuncLit:
		t.symbolTable = t.symbolTable.Fork(false)
		defer func() { t.symbolTable = t.symbolTable.Parent(true) }()

		// function((param1), (param2), ...)
		//   (body)
		// end

		var params []string
		for _, p := range node.Type.Params.List {
			t.symbolTable.Define(p.Name)

			param, err := t.convert(p)
			if err != nil {
				return "", err
			}
			params = append(params, param)
		}

		out := t.line("function(%s)", strings.Join(params, ","))

		t.indentLevel++

		body, err := t.convert(node.Body)
		if err != nil {
			return "", err
		}
		out += body

		t.indentLevel--
		out += t.line("end")

		return out, nil

	case *ast.ImportExpr:
		return "", t.error(node, "import expression not supported")

	case *ast.ExportStmt:
		return "", t.error(node, "export statement not supported")

	case *ast.ErrorExpr:
		return "", t.error(node, "error expression not supported")

	case *ast.ImmutableExpr:
		// TODO: use metamethods (http://lua-users.org/wiki/ReadOnlyTables)
		return "", t.error(node, "immutable expression not supported")

	case *ast.CondExpr:
		// (function() if (cond) then return (true-expr) else return (false-expr) end end)()

		cond, err := t.convert(node.Cond)
		if err != nil {
			return "", err
		}

		trueExpr, err := t.convert(node.True)
		if err != nil {
			return "", err
		}

		falseExpr, err := t.convert(node.False)
		if err != nil {
			return "", err
		}

		return "(function() if (" + cond + ") then return (" + trueExpr + ") else return (" + falseExpr + ") end end)()", nil
	}

	return "", nil
}

func (t *Transpiler) line(format string, args ...interface{}) string {
	return strings.Repeat(t.options.Indent, t.indentLevel) +
		fmt.Sprintf(format, args...) + "\n"
}

func (t *Transpiler) convertAssignment(node ast.Node, lhs, rhs []ast.Expr, op token.Token) (string, error) {
	numLHS, numRHS := len(lhs), len(rhs)
	if numLHS > 1 || numRHS > 1 {
		return "", t.error(node, "tuple assignment not allowed")
	}

	// resolve and compile left-hand side
	ident, selectors := resolveAssignLHS(lhs[0])
	numSel := len(selectors)

	if op == token.Define && numSel > 0 {
		// using selector on new variable does not make sense
		return "", t.error(node, "operator ':=' not allowed with selector")
	}

	symbol, depth, exists := t.symbolTable.Resolve(ident)
	if op == token.Define {
		if depth == 0 && exists {
			return "", t.error(node, "'%s' redeclared in this block", ident)
		}

		if reservedVarName.MatchString(ident) {
			return "", t.error(node, "cannot use variable name '%s'", ident)
		}

		symbol = t.symbolTable.Define(ident)
	} else {
		if !exists {
			return "", t.error(node, "unresolved reference '%s'", ident)
		}
	}

	// left-hand side
	left, err := t.convert(lhs[0])
	if err != nil {
		return "", err
	}

	// right-hand side
	right, err := t.convert(rhs[0])
	if err != nil {
		return "", err
	}

	switch op {
	case token.Define:
		pref := ""
		if !t.options.EnableGlobalScope || symbol.Scope != compiler.ScopeGlobal {
			pref = "local "
		}
		return pref + left + "=" + right, nil
	case token.Assign:
		return left + "=" + right, nil
	case token.AddAssign:
		t.helpersUsed[helperStringConcat] = true
		return left + "=" + left + "+" + right, nil
	case token.SubAssign:
		return left + "=" + left + "-" + right, nil
	case token.MulAssign:
		return left + "=" + left + "*" + right, nil
	case token.QuoAssign:
		return left + "=" + left + "/" + right, nil
	case token.RemAssign:
		return left + "=" + left + "%" + right, nil
	case token.AndAssign, token.OrAssign, token.AndNotAssign, token.XorAssign, token.ShlAssign, token.ShrAssign:
		// TODO: bitwise operators
		return "", t.error(node, "compound assignment "+op.String()+" not supported")
	default:
		return "", t.error(node, "assignment operator "+op.String()+"not supported")
	}
}

func (t *Transpiler) continueVarName() string {
	return fmt.Sprintf("__cont_%d__", t.loopDepth)
}

func resolveAssignLHS(expr ast.Expr) (name string, selectors []ast.Expr) {
	switch term := expr.(type) {
	case *ast.SelectorExpr:
		name, selectors = resolveAssignLHS(term.Expr)
		selectors = append(selectors, term.Sel)
		return

	case *ast.IndexExpr:
		name, selectors = resolveAssignLHS(term.Expr)
		selectors = append(selectors, term.Index)

	case *ast.Ident:
		name = term.Name
	}

	return
}

func (t *Transpiler) error(node ast.Node, format string, args ...interface{}) error {
	return &Error{
		fileSet: t.file.Set(),
		node:    node,
		error:   fmt.Errorf(format, args...),
	}
}

func (t *Transpiler) parse() (*ast.File, error) {
	p := parser.NewParser(t.file, t.src, nil)
	return p.ParseFile()
}
