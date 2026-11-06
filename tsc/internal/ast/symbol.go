package ast

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/microsoft/TypeScript/tsc/internal/json"
)

// SymbolName is an interned name used for symbol identity and lookup.
// Its zero value is EmptySymbolName.
//
// The low bits hold an intern table ID; the upper bits encode derived names
// and tag internal names. See symbolname_derived.go.
//
// Fixed names, such as SymbolNamePrototype, are constants generated from the
// list in tools/scripts/tsc/symbolNames.ts by `npx hereby generate:ast`.
type SymbolName uint64

func MakeSymbolName(text string) SymbolName {
	return symbolNames.make(text, hashSymbolNameText(text))
}

func (name SymbolName) String() string {
	if name.derivedKind() != derivedNamePlain {
		return name.derivedValue()
	}
	return symbolNames.text(uint32(name))
}

// MarshalText uses the escaped spelling so internal names remain valid UTF-8.
func (name SymbolName) MarshalText() ([]byte, error) {
	text := EscapeSymbolName(name)
	if !utf8.ValidString(text) {
		return nil, errors.New("symbol name contains invalid UTF-8")
	}
	return []byte(text), nil
}

func (name *SymbolName) UnmarshalText(text []byte) error {
	if !utf8.Valid(text) {
		return errors.New("symbol name contains invalid UTF-8")
	}
	*name = UnescapeSymbolName(string(text))
	return nil
}

func (name *SymbolName) UnmarshalJSON(data []byte) error {
	if !utf8.Valid(data) {
		return errors.New("symbol name contains invalid UTF-8")
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	return name.UnmarshalText([]byte(text))
}


const InternalSymbolNamePrefix = "\xFE" // Invalid UTF8 sequence, will never occur as IdentifierName

// SymbolNameText returns a display name, preserving private identifier spelling.
func SymbolNameText(symbol Symbol) string {
	if !symbol.ValueDeclaration().IsNil() && IsPrivateIdentifierClassElementDeclaration(symbol.ValueDeclaration()) {
		return symbol.ValueDeclaration().Name().Text()
	}
	return symbol.Name().String()
}

// EscapeAllInternalSymbolNames replaces internal symbol name markers ("\xFE") with "__".
func EscapeAllInternalSymbolNames(name string) string {
	return strings.ReplaceAll(name, InternalSymbolNamePrefix, "__")
}

func EscapeInternalSymbolName(symbolName SymbolName) string {
	name := symbolName.String()
	if rest, ok := strings.CutPrefix(name, InternalSymbolNamePrefix); ok {
		return "__" + rest
	}
	return name
}

// EscapeSymbolName converts a binder symbol name into its escaped "__String"
// form. Internal names (prefixed with the "\xFE" sentinel) become "__"-prefixed,
// and user names that already begin with "__" gain an extra leading underscore
// so they can be distinguished from internal names.
func EscapeSymbolName(symbolName SymbolName) string {
	name := symbolName.String()
	if rest, ok := strings.CutPrefix(name, InternalSymbolNamePrefix); ok {
		return "__" + rest
	}
	if len(name) >= 2 && name[0] == '_' && name[1] == '_' {
		return "_" + name
	}
	return name
}

// UnescapeSymbolName reverses EscapeSymbolName and interns the decoded name.
func UnescapeSymbolName(name string) SymbolName {
	if strings.HasPrefix(name, "___") {
		name = name[1:]
	} else if rest, ok := strings.CutPrefix(name, "__"); ok {
		name = InternalSymbolNamePrefix + rest
	}
	return MakeSymbolName(name)
}
