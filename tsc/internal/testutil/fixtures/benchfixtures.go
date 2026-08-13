package fixtures

import (
	"path/filepath"

	"github.com/microsoft/TypeScript/tsc/internal/repo"
	"github.com/microsoft/TypeScript/tsc/internal/testutil/filefixture"
)

var BenchFixtures = []filefixture.Fixture{
	filefixture.FromString("empty.ts", "empty.ts", ""),
	filefixture.FromFile("checker.ts", filepath.Join(repo.LegacyReferencePath(), "src/compiler/checker.ts")),
	filefixture.FromFile("dom.generated.d.ts", filepath.Join(repo.LegacyReferencePath(), "src/lib/dom.generated.d.ts")),
	filefixture.FromFile("Herebyfile.mjs", filepath.Join(repo.LegacyReferencePath(), "Herebyfile.mjs")),
	filefixture.FromFile("jsxComplexSignatureHasApplicabilityError.tsx", filepath.Join(repo.LegacyReferencePath(), "tests/cases/compiler/jsxComplexSignatureHasApplicabilityError.tsx")),
}
