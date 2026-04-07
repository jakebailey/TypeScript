package core

//go:generate go -C ../../../tools tool stringer -dir=../tsc/internal/core -type=LanguageVariant -output=languagevariant_stringer_generated.go
//go:generate npx dprint fmt languagevariant_stringer_generated.go

type LanguageVariant int32

const (
	LanguageVariantStandard LanguageVariant = iota
	LanguageVariantJSX
)
