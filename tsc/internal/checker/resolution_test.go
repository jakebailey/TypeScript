package checker

import "testing"

func TestResolutionStackQueryIgnoresResolutionStart(t *testing.T) {
	t.Parallel()

	signature := &Signature{}
	c := &Checker{
		typeResolutions: []TypeResolution{{
			target:       signature,
			propertyName: TypeSystemPropertyNameResolvedReturnType,
			result:       true,
		}},
		resolutionStart: 1,
	}

	if !c.isInResolutionStack(signature, TypeSystemPropertyNameResolvedReturnType) {
		t.Fatal("isInResolutionStack() missed an entry below resolutionStart")
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
		if !c.isInResolutionStack(target, TypeSystemPropertyNameResolvedReturnType) {
			b.Fatal("resolution stack query missed an entry below resolutionStart")
		}
	}
}
