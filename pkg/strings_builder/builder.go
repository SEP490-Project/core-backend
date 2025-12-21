// Package stringsbuilder provides utilities for building and manipulating strings efficiently.
package stringsbuilder

import (
	"fmt"
	"strings"
)

// region: ===================== String Builder with Method Chaining =====================

// Builder is a wrapper around strings.Builder that supports method chaining.
type Builder struct {
	builder strings.Builder
}

// NewStringBuilder creates a new instance with optional pre-allocation.
func NewStringBuilder(capacity ...int) *Builder {
	sb := &Builder{}
	if len(capacity) > 0 && capacity[0] > 0 {
		sb.builder.Grow(capacity[0])
	}
	return sb
}

// Append adds a string to the builder and returns the instance for chaining.
func (sb *Builder) Append(s string) *Builder {
	sb.builder.WriteString(s)
	return sb
}

// If starts a conditional chain.
// It returns a *ConditionalBuilder instead of *StringBuilder.
func (sb *Builder) If(condition bool, s string) *ConditionalBuilder {
	// Execute immediately if true
	if condition {
		sb.builder.WriteString(s)
	}
	// Return the wrapper with the state of this condition
	return &ConditionalBuilder{
		parent:    sb,
		isMatched: condition,
	}
}

// IfFunc allows complex logic inside a block.
func (sb *Builder) IfFunc(condition bool, fn func(*Builder)) *ConditionalBuilder {
	if condition {
		fn(sb)
	}
	return &ConditionalBuilder{
		parent:    sb,
		isMatched: condition,
	}
}

// AppendLine adds a string followed by a new line and returns the instance.
func (sb *Builder) AppendLine(s string) *Builder {
	sb.builder.WriteString(s)
	sb.builder.WriteByte('\n')
	return sb
}

// AppendFormat adds a formatted string (like Sprintf) and returns the instance.
func (sb *Builder) AppendFormat(format string, args ...any) *Builder {
	fmt.Fprintf(&sb.builder, format, args...)
	return sb
}

func (sb *Builder) AppendLineFormat(format string, args ...any) *Builder {
	sb.AppendFormat(format, args...)
	sb.builder.WriteByte('\n')
	return sb
}

// AppendRune adds a single rune (character) and returns the instance.
func (sb *Builder) AppendRune(r rune) *Builder {
	sb.builder.WriteRune(r)
	return sb
}

// AppendInt adds an integer as a string and returns the instance.
func (sb *Builder) AppendInt(i int) *Builder {
	fmt.Fprintf(&sb.builder, "%d", i)
	return sb
}

// String returns the accumulated string.
func (sb *Builder) String() string {
	return sb.builder.String()
}

// Len returns the number of accumulated bytes.
func (sb *Builder) Len() int {
	return sb.builder.Len()
}

// Reset clears the builder to be reused.
func (sb *Builder) Reset() *Builder {
	sb.builder.Reset()
	return sb
}

// endregion

// region: ===================== Conditional String Builder =====================

type ConditionalBuilder struct {
	parent    *Builder
	isMatched bool // true if any condition in the chain has been met
}

// ElseIf checks a new condition only if the previous ones were false.
func (b *ConditionalBuilder) ElseIf(condition bool, s string) *ConditionalBuilder {
	if b.isMatched {
		return b // Already matched a previous If/ElseIf, skip this.
	}

	if condition {
		b.parent.builder.WriteString(s)
		b.isMatched = true
	}
	return b
}

// Else adds content if none of the previous conditions were met.
func (b *ConditionalBuilder) Else(s string) *ConditionalBuilder {
	if !b.isMatched {
		b.parent.builder.WriteString(s)
		b.isMatched = true // Mark as matched so subsequent logic knows we are done
	}
	return b
}

// ElseFunc allows complex logic in the Else block.
func (b *ConditionalBuilder) ElseFunc(fn func(*Builder)) *ConditionalBuilder {
	if !b.isMatched {
		fn(b.parent)
		b.isMatched = true
	}
	return b
}

// End finishes the conditional block and returns the original StringBuilder
// to allow continued chaining.
func (b *ConditionalBuilder) End() *Builder {
	return b.parent
}

// endregion
