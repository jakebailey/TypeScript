package fourslash_test

import (
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/fourslash"
	"github.com/microsoft/TypeScript/tsc/internal/testutil"
)

func TestFormatNotEqualsEqualsEquals(t *testing.T) {
	t.Parallel()
	defer testutil.RecoverAndFail(t, "Panic on fourslash test")
	const content = `/**/const malformed = 1 !===    2;
const valid = 1 ==    2 && 1 !=    2 && 1 ===    2 && 1 !==    2;
const repeated = 1 !===    2 !===    3;`
	const expected = `const malformed = 1 !=== 2;
const valid = 1 == 2 && 1 != 2 && 1 === 2 && 1 !== 2;
const repeated = 1 !=== 2 !=== 3;`
	f, done := fourslash.NewFourslash(t, nil /*capabilities*/, content)
	defer done()

	f.FormatDocument(t, "")
	f.VerifyCurrentFileContent(t, expected)
	f.FormatDocument(t, "")
	f.VerifyCurrentFileContent(t, expected)
}
