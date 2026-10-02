package fourslash_test

import (
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/fourslash"
	"github.com/microsoft/TypeScript/tsc/internal/testutil"
)

func TestRecursiveFunctionQuickInfo(t *testing.T) {
	t.Parallel()
	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	const content = `
const arrow/*arrow*/ = () => arrow;
const broad: unknown = function self/*self*/() { return self; };
arrow/*use*/()();
const object/*object*/ = { next: () => object };
const tuple/*tuple*/ = [() => tuple] as const;
`
	f, done := fourslash.NewFourslash(t, nil, content)
	defer done()
	f.VerifyQuickInfoAt(t, "arrow", "const arrow: () => typeof arrow", "")
	f.VerifyQuickInfoAt(t, "use", "const arrow: () => () => typeof arrow", "")
	f.VerifyQuickInfoAt(t, "self", "function self(): () => typeof self", "")
	f.VerifyQuickInfoAt(t, "object", "const object: {\n    next: () => typeof object;\n}", "")
	f.VerifyQuickInfoAt(t, "tuple", "const tuple: readonly [() => ...]", "")
	f.VerifyNoErrors(t)
}
