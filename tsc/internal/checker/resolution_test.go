package checker

import "testing"

func TestResolutionStackQueryIgnoresResolutionStart(t *testing.T) {
	signature := &Signature{}
	c := &Checker{
		typeResolutions: []TypeResolution{{
			target:       signature,
			propertyName: TypeSystemPropertyNameResolvedReturnType,
			result:       true,
		}},
		resolutionStart: 1,
	}

	if got := c.findResolutionCycleStartIndex(signature, TypeSystemPropertyNameResolvedReturnType, 0); got != 0 {
		t.Fatalf("findResolutionCycleStartIndex() = %d, want 0", got)
	}
	if !c.isResolvingReturnTypeOfSignature(signature) {
		t.Fatal("isResolvingReturnTypeOfSignature() missed an entry below resolutionStart")
	}
	if !c.pushTypeResolution(signature, TypeSystemPropertyNameResolvedReturnType) {
		t.Fatal("pushTypeResolution() detected a cycle below resolutionStart")
	}
}

func BenchmarkResolutionStackQueryAcrossReset(b *testing.B) {
	const stackDepth = 64
	signatures := make([]Signature, stackDepth)
	resolutions := make([]TypeResolution, stackDepth)
	for i := range resolutions {
		resolutions[i] = TypeResolution{
			target:       &signatures[i],
			propertyName: TypeSystemPropertyNameResolvedReturnType,
			result:       true,
		}
	}
	c := &Checker{
		typeResolutions: resolutions,
		resolutionStart: stackDepth,
	}
	target := &signatures[0]

	b.ResetTimer()
	for b.Loop() {
		if c.findResolutionCycleStartIndex(target, TypeSystemPropertyNameResolvedReturnType, 0) != 0 {
			b.Fatal("resolution stack query missed an entry below resolutionStart")
		}
	}
}
