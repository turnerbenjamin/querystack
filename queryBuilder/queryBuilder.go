package querybuilder

import (
	"fmt"
	"strings"
)

type Builder struct {
	sb   *strings.Builder
	args []any
}

// arg adds a new argument to the args list and returns a unique placeholder for
// use in the sql statement
func (b *Builder) Placeholder(v any) string {
	if b.args == nil {
		b.args = make([]any, 0, 8)
	}
	b.args = append(b.args, v)
	return fmt.Sprintf("@p%d", len(b.args))
}

func (b *Builder) Write(statement string, args ...any) {
	if b.sb == nil {
		b.sb = &strings.Builder{}
	}
	fmt.Fprintf(b.sb, statement, args...)
}

func (b *Builder) String() string {
	return b.sb.String()
}

func (b *Builder) Args() []any {
	return b.args
}
