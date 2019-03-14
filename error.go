package tengo2lua

import (
	"fmt"

	"github.com/d5/tengo/compiler/ast"
	"github.com/d5/tengo/compiler/source"
)

type Error struct {
	fileSet *source.FileSet
	node    ast.Node
	error   error
}

func (e *Error) Error() string {
	filePos := e.fileSet.Position(e.node.Pos())
	return fmt.Sprintf("Transpile Error: %s\n\tat %s", e.error.Error(), filePos)
}
