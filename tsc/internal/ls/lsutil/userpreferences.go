package lsutil

import (
	"maps"
	"strings"
	"sync"

	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/json"
	"github.com/microsoft/TypeScript/tsc/internal/modulespecifiers"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/vfsmatch"
)

func NewDefaultUserPreferences() UserPreferences {
	return UserPreferences{
		FormatCodeSettings: GetDefaultFormatCodeSettings(),

		IncludeCompletionsForModuleExports:    core.TSTrue,
		IncludeCompletionsForImportStatements: core.TSTrue,
		EnableAutoClosingTags:                 core.TSTrue,
		EnableJSDocCompletions:                core.TSTrue,
		GenerateReturnInDocTemplate:           core.TSTrue,

		AllowRenameOfImportPath:            core.TSTrue,
		ProvideRefactorNotApplicableReason: core.TSTrue,
		EnableFormatting:                   core.TSTrue,
		EnableValidation:                   core.TSTrue,
		DisplayPartsForJSDoc:               core.TSTrue,
		DisableLineTextInReferences:        core.TSTrue,
		ReportStyleChecksAsWarnings:        core.TSTrue,

		ExcludeLibrarySymbolsInNavTo: core.TSTrue,
		WorkspaceSymbolsScope:        WorkspaceSymbolsScopeAllOpenProjects,
	}
}

// UserPreferences represents TypeScript language service preferences.
//
// Fields are populated using two tags:
//   - `raw:"name"` or `raw:"name,invert"` - TypeScript/raw name for unstable section lookup
//   - `config:"path.to.setting"` or `config:"path.to.setting,invert"` - VS Code nested config path
//
// At least one tag must be present on each preference field.
// The `,invert` modifier inverts boolean values (e.g., VS Code's "suppress" -> our "include").
type UserPreferences struct {
	FormatCodeSettings FormatCodeSettings

	QuotePreference                           QuotePreference `raw:"quotePreference" config:"preferences.quoteStyle"`
	LazyConfiguredProjectsFromExternalProject core.Tristate   `raw:"lazyConfiguredProjectsFromExternalProject"` // !!!

	// A positive integer indicating the maximum length of a hover text before it is truncated.
	//
	// Default: `500`
	MaximumHoverLength int `raw:"maximumHoverLength"` // !!!

	// ------- Completions -------

	// If enabled, TypeScript will search through all external modules' exports and add them to the completions list.
	// This affects lone identifier completions but not completions on the right hand side of `obj.`.
	IncludeCompletionsForModuleExports core.Tristate `raw:"includeCompletionsForModuleExports" config:"suggest.autoImports"`
	// Enables auto-import-style completions on partially-typed import statements. E.g., allows
	// `import write|` to be completed to `import { writeFile } from "fs"`.
	IncludeCompletionsForImportStatements core.Tristate `raw:"includeCompletionsForImportStatements" config:"suggest.includeCompletionsForImportStatements"`
	// Unless this option is `false`,  member completion lists triggered with `.` will include entries
	// on potentially-null and potentially-undefined values, with insertion text to replace
	// preceding `.` tokens with `?.`.
	IncludeAutomaticOptionalChainCompletions core.Tristate `raw:"includeAutomaticOptionalChainCompletions" config:"suggest.includeAutomaticOptionalChainCompletions"`
	// If enabled, completions for class members (e.g. methods and properties) will include
	// a whole declaration for the member.
	// E.g., `class A { f| }` could be completed to `class A { foo(): number {} }`, instead of
	// `class A { foo }`.
	IncludeCompletionsWithClassMemberSnippets core.Tristate `raw:"includeCompletionsWithClassMemberSnippets" config:"suggest.classMemberSnippets.enabled"`
	// If enabled, object literal methods will have a method declaration completion entry in addition
	// to the regular completion entry containing just the method name.
	// E.g., `const objectLiteral: T = { f| }` could be completed to `const objectLiteral: T = { foo(): void {} }`,
	// in addition to `const objectLiteral: T = { foo }`.
	IncludeCompletionsWithObjectLiteralMethodSnippets core.Tristate               `raw:"includeCompletionsWithObjectLiteralMethodSnippets" config:"suggest.objectLiteralMethodSnippets.enabled"`
	JsxAttributeCompletionStyle                       JsxAttributeCompletionStyle `raw:"jsxAttributeCompletionStyle" config:"preferences.jsxAttributeCompletionStyle"`
	EnableAutoClosingTags                             core.Tristate               `raw:"autoClosingTags" config:"autoClosingTags.enabled" fallbackConfig:"autoClosingTags"`
	EnableJSDocCompletions                            core.Tristate               `raw:"completeJSDocs" config:"suggest.jsdoc.enabled" fallbackConfig:"suggest.completeJSDocs"`
	GenerateReturnInDocTemplate                       core.Tristate               `raw:"generateReturnInDocTemplate" config:"suggest.jsdoc.generateReturns"`

	// ------- AutoImports --------

	ImportModuleSpecifierPreference modulespecifiers.ImportModuleSpecifierPreference `raw:"importModuleSpecifierPreference" config:"preferences.importModuleSpecifier"` // !!!
	// Determines whether we import `foo/index.ts` as "foo", "foo/index", or "foo/index.js"
	ImportModuleSpecifierEnding         modulespecifiers.ImportModuleSpecifierEndingPreference `raw:"importModuleSpecifierEnding" config:"preferences.importModuleSpecifierEnding"`             // !!!
	AutoImportSpecifierExcludeRegexes   []string                                               `raw:"autoImportSpecifierExcludeRegexes" config:"preferences.autoImportSpecifierExcludeRegexes"` // !!!
	AutoImportFileExcludePatterns       []string                                               `raw:"autoImportFileExcludePatterns" config:"preferences.autoImportFileExcludePatterns"`
	AutoImportEntrypointDirectorySearch core.Tristate                                          `raw:"autoImportEntrypointDirectorySearch" config:"preferences.autoImportEntrypointDirectorySearch"`
	PreferTypeOnlyAutoImports           core.Tristate                                          `raw:"preferTypeOnlyAutoImports" config:"preferences.preferTypeOnlyAutoImports"`

	// ------- OrganizeImports -------

	// Indicates which deterministic preset should be used to sort imports.
	// "auto" detects the existing ordinal case sensitivity where possible.
	OrganizeImportsSort OrganizeImportsSort `raw:"organizeImportsSort" config:"preferences.organizeImports.sort"` // !!!
	// Indicates whether imports should be organized in a case-insensitive manner.
	//
	// Default: TSUnknown ("auto" in strada), will perform detection
	OrganizeImportsIgnoreCase core.Tristate `raw:"organizeImportsIgnoreCase" config:"preferences.organizeImports.caseSensitivity"` // !!!
	// Indicates whether imports should be organized via an "ordinal" (binary) comparison using the numeric value of their
	// code points, or via "unicode" natural sorting. This implementation is locale-agnostic and approximates the practical
	// import-sorting behavior rather than the full Unicode Collation Algorithm.
	//
	// Default: Ordinal
	OrganizeImportsCollation OrganizeImportsCollation `raw:"organizeImportsCollation" config:"preferences.organizeImports.unicodeCollation"` // !!!
	// Indicates the locale to use for "unicode" collation in legacy clients. This is accepted for compatibility, but
	// currently ignored because organize-import sorting is deterministic and locale-agnostic.
	//
	// This preference is ignored if organizeImportsCollation is not `unicode`.
	//
	// Default: `"en"`
	OrganizeImportsLocale string `raw:"organizeImportsLocale" config:"preferences.organizeImports.locale"` // !!!
	// Indicates whether numeric collation should be used for digit sequences in strings. When `true`, will collate
	// strings such that `a1z < a2z < a100z`. When `false`, will collate strings such that `a1z < a100z < a2z`.
	//
	// This preference is ignored if organizeImportsCollation is not `unicode`.
	//
	// Default: `false`
	OrganizeImportsNumericCollation core.Tristate `raw:"organizeImportsNumericCollation" config:"preferences.organizeImports.numericCollation"` // !!!
	// Indicates whether accents and other diacritic marks are considered unequal for the purpose of sorting.
	//
	// This preference is ignored if organizeImportsCollation is not `unicode`.
	//
	// Default: `true`
	OrganizeImportsAccentCollation core.Tristate `raw:"organizeImportsAccentCollation" config:"preferences.organizeImports.accentCollation"` // !!!
	// Indicates whether upper case or lower case should sort first.
	//
	// This permission is ignored if:
	//	- organizeImportsCollation is not `unicode`
	//	- organizeImportsIgnoreCase is `true`
	//	- organizeImportsIgnoreCase is `auto` and the auto-detected case sensitivity is case-insensitive.
	//
	// Default: `false`
	OrganizeImportsCaseFirst OrganizeImportsCaseFirst `raw:"organizeImportsCaseFirst" config:"preferences.organizeImports.caseFirst"` // !!!
	// Indicates where named type-only imports should sort. "inline" sorts named imports without regard to if the import is type-only.
	//
	// Default: `auto`, which defaults to `last`
	OrganizeImportsTypeOrder OrganizeImportsTypeOrder `raw:"organizeImportsTypeOrder" config:"preferences.organizeImports.typeOrder"` // !!!

	// ------- MoveToFile -------

	AllowTextChangesInNewFiles core.Tristate `raw:"allowTextChangesInNewFiles"` // !!!

	// ------- Rename -------

	UseAliasesForRename     core.Tristate `raw:"providePrefixAndSuffixTextForRename" config:"preferences.useAliasesForRenames"`
	AllowRenameOfImportPath core.Tristate `raw:"allowRenameOfImportPath"`

	// ------- CodeFixes/Refactors -------

	ProvideRefactorNotApplicableReason core.Tristate `raw:"provideRefactorNotApplicableReason"` // !!!

	// ------- InlayHints -------

	InlayHints InlayHintsPreferences

	// ------- CodeLens -------

	CodeLens CodeLensUserPreferences

	// ------- Definition -------

	PreferGoToSourceDefinition bool `raw:"preferGoToSourceDefinition"`

	// ------- Symbols -------

	ExcludeLibrarySymbolsInNavTo core.Tristate         `raw:"excludeLibrarySymbolsInNavTo" config:"workspaceSymbols.excludeLibrarySymbols"`
	WorkspaceSymbolsScope        WorkspaceSymbolsScope `config:"workspaceSymbols.scope"`

	// ------- Misc -------

	EnableFormatting            core.Tristate `raw:"formatEnabled" config:"format.enabled" fallbackConfig:"format.enable"`
	EnableValidation            core.Tristate `raw:"validateEnabled" config:"validate.enabled" fallbackConfig:"validate.enable"`
	DisableSuggestions          core.Tristate `raw:"disableSuggestions"`          // !!!
	DisableLineTextInReferences core.Tristate `raw:"disableLineTextInReferences"` // !!!
	DisplayPartsForJSDoc        core.Tristate `raw:"displayPartsForJSDoc"`        // !!!
	ReportStyleChecksAsWarnings core.Tristate `raw:"reportStyleChecksAsWarnings" config:"reportStyleChecksAsWarnings"`
	Locale                      string        `config:"locale"`

	// ------- ATA -------

	// DisableAutomaticTypeAcquisition is the deprecated setting from typescript.disableAutomaticTypeAcquisition.
	DisableAutomaticTypeAcquisition core.Tristate `raw:"disableAutomaticTypeAcquisition" config:"disableAutomaticTypeAcquisition"`
	// AutomaticTypeAcquisitionEnabled is the unified setting from tsserver.automaticTypeAcquisition.enabled under the js/ts section.
	// When set, it takes precedence over DisableAutomaticTypeAcquisition.
	AutomaticTypeAcquisitionEnabled core.Tristate `raw:"automaticTypeAcquisitionEnabled" config:"tsserver.automaticTypeAcquisition.enabled"`
	// TODO: add tsserver.web.typeAcquisition.enabled under the js/ts section for the web variant when web support is implemented.

	// ------- Project Configuration -------

	// CustomConfigFileName specifies a custom config file name to use before defaulting to tsconfig.json/jsconfig.json.
	CustomConfigFileName string `raw:"customConfigFileName" config:"customConfigFileName"`
}

// IsATADisabled returns whether Automatic Type Acquisition is disabled based on user preferences.
// It checks the unified setting (tsserver.automaticTypeAcquisition.enabled) first,
// then falls back to the deprecated setting (disableAutomaticTypeAcquisition).
func (p UserPreferences) IsATADisabled() bool {
	if !p.AutomaticTypeAcquisitionEnabled.IsUnknown() {
		return !p.AutomaticTypeAcquisitionEnabled.IsTrue()
	}
	return p.DisableAutomaticTypeAcquisition.IsTrue()
}

type InlayHintsPreferences struct {
	IncludeInlayParameterNameHints                        IncludeInlayParameterNameHints `raw:"includeInlayParameterNameHints" config:"inlayHints.parameterNames.enabled"`
	IncludeInlayParameterNameHintsWhenArgumentMatchesName core.Tristate                  `raw:"includeInlayParameterNameHintsWhenArgumentMatchesName" config:"inlayHints.parameterNames.suppressWhenArgumentMatchesName,invert"`
	IncludeInlayFunctionParameterTypeHints                core.Tristate                  `raw:"includeInlayFunctionParameterTypeHints" config:"inlayHints.parameterTypes.enabled"`
	IncludeInlayVariableTypeHints                         core.Tristate                  `raw:"includeInlayVariableTypeHints" config:"inlayHints.variableTypes.enabled"`
	IncludeInlayVariableTypeHintsWhenTypeMatchesName      core.Tristate                  `raw:"includeInlayVariableTypeHintsWhenTypeMatchesName" config:"inlayHints.variableTypes.suppressWhenTypeMatchesName,invert"`
	IncludeInlayPropertyDeclarationTypeHints              core.Tristate                  `raw:"includeInlayPropertyDeclarationTypeHints" config:"inlayHints.propertyDeclarationTypes.enabled"`
	IncludeInlayFunctionLikeReturnTypeHints               core.Tristate                  `raw:"includeInlayFunctionLikeReturnTypeHints" config:"inlayHints.functionLikeReturnTypes.enabled"`
	IncludeInlayEnumMemberValueHints                      core.Tristate                  `raw:"includeInlayEnumMemberValueHints" config:"inlayHints.enumMemberValues.enabled"`
}

type CodeLensUserPreferences struct {
	ReferencesCodeLensEnabled                     core.Tristate `raw:"referencesCodeLensEnabled" config:"referencesCodeLens.enabled"`
	ImplementationsCodeLensEnabled                core.Tristate `raw:"implementationsCodeLensEnabled" config:"implementationsCodeLens.enabled"`
	ReferencesCodeLensShowOnAllFunctions          core.Tristate `raw:"referencesCodeLensShowOnAllFunctions" config:"referencesCodeLens.showOnAllFunctions"`
	ImplementationsCodeLensShowOnInterfaceMethods core.Tristate `raw:"implementationsCodeLensShowOnInterfaceMethods" config:"implementationsCodeLens.showOnInterfaceMethods"`
	ImplementationsCodeLensShowOnAllClassMethods  core.Tristate `raw:"implementationsCodeLensShowOnAllClassMethods" config:"implementationsCodeLens.showOnAllClassMethods"`
}

// --- Enum Types ---

type QuotePreference string

type WorkspaceSymbolsScope string

const (
	WorkspaceSymbolsScopeAllOpenProjects WorkspaceSymbolsScope = "allOpenProjects"
	WorkspaceSymbolsScopeCurrentProject  WorkspaceSymbolsScope = "currentProject"
)

const (
	QuotePreferenceUnknown QuotePreference = ""
	QuotePreferenceAuto    QuotePreference = "auto"
	QuotePreferenceDouble  QuotePreference = "double"
	QuotePreferenceSingle  QuotePreference = "single"
)

type JsxAttributeCompletionStyle string

const (
	JsxAttributeCompletionStyleUnknown JsxAttributeCompletionStyle = ""
	JsxAttributeCompletionStyleAuto    JsxAttributeCompletionStyle = "auto"
	JsxAttributeCompletionStyleBraces  JsxAttributeCompletionStyle = "braces"
	JsxAttributeCompletionStyleNone    JsxAttributeCompletionStyle = "none"
)

type IncludeInlayParameterNameHints string

const (
	IncludeInlayParameterNameHintsNone     IncludeInlayParameterNameHints = ""
	IncludeInlayParameterNameHintsAll      IncludeInlayParameterNameHints = "all"
	IncludeInlayParameterNameHintsLiterals IncludeInlayParameterNameHints = "literals"
)

type OrganizeImportsSort int

const (
	OrganizeImportsSortAuto OrganizeImportsSort = iota
	OrganizeImportsSortOrdinal
	OrganizeImportsSortOrdinalIgnoreCase
	OrganizeImportsSortNatural
	OrganizeImportsSortNaturalIgnoreCase
)

type OrganizeImportsCollation bool

const (
	OrganizeImportsCollationOrdinal OrganizeImportsCollation = false
	OrganizeImportsCollationUnicode OrganizeImportsCollation = true
)

type OrganizeImportsCaseFirst int

const (
	OrganizeImportsCaseFirstFalse OrganizeImportsCaseFirst = 0
	OrganizeImportsCaseFirstLower OrganizeImportsCaseFirst = 1
	OrganizeImportsCaseFirstUpper OrganizeImportsCaseFirst = 2
)

type OrganizeImportsTypeOrder int

const (
	OrganizeImportsTypeOrderAuto   OrganizeImportsTypeOrder = 0
	OrganizeImportsTypeOrderLast   OrganizeImportsTypeOrder = 1
	OrganizeImportsTypeOrderInline OrganizeImportsTypeOrder = 2
	OrganizeImportsTypeOrderFirst  OrganizeImportsTypeOrder = 3
)

// --- Generated preference parsing infrastructure ---

// configPathParsers provides field-specific config value parsers that override the default
// type-based parser when the VS Code config value format differs from the Go field type.
var configPathParsers = map[string]func(any) any{
	// VS Code sends caseSensitivity as a string ("auto"/"caseSensitive"/"caseInsensitive"),
	// but OrganizeImportsIgnoreCase is a core.Tristate.
	"preferences.organizeImports.caseSensitivity": func(val any) any {
		if s, ok := val.(string); ok {
			switch strings.ToLower(s) {
			case "caseinsensitive":
				return core.TSTrue
			case "casesensitive":
				return core.TSFalse
			}
		}
		if b, ok := val.(bool); ok {
			if b {
				return core.TSTrue
			}
			return core.TSFalse
		}
		return core.TSUnknown
	},
}

type fieldInfo struct {
	rawName             string // raw name for unstable section lookup (e.g., "quotePreference")
	configPath          string // dotted path for config (e.g., "preferences.quoteStyle")
	fallbackConfigPaths []configPathInfo
	rawInvert           bool // whether to invert boolean values for raw name
	configInvert        bool // whether to invert boolean values for config path
	set                 func(*UserPreferences, any)
	serialize           func(*UserPreferences) any
	merge               func(*UserPreferences, UserPreferences)
}

type configPathInfo struct {
	path   string
	invert bool
}

var fieldInfoCache = sync.OnceValue(func() []fieldInfo {
	return generatedUserPreferenceFieldInfos()
})

// unstableNameIndex maps raw names to fieldInfo index for unstable section lookup.
var unstableNameIndex = sync.OnceValue(func() map[string]int {
	infos := fieldInfoCache()
	index := make(map[string]int, len(infos))
	for i, info := range infos {
		if info.rawName != "" {
			index[info.rawName] = i
		}
	}
	return index
})

func getNestedValue(config map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	current := any(config)
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func setNestedValue(config map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := config
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}

func setRawFieldsFromConfig(p *UserPreferences, infos []fieldInfo, settings map[string]any) {
	index := unstableNameIndex()
	for name, value := range settings {
		if idx, found := index[name]; found {
			info := infos[idx]
			if info.rawInvert {
				if b, ok := value.(bool); ok {
					value = !b
				}
			}
			info.set(p, value)
		}
	}
}

func (p UserPreferences) withConfig(config map[string]any) UserPreferences {
	infos := fieldInfoCache()

	// Raw UserPreferences can be provided directly, notably via LSP initializationOptions.
	setRawFieldsFromConfig(&p, infos, config)

	// Process "unstable" section first - allows any field to be set by raw name.
	// This mirrors VS Code's behavior: { ...config.get('unstable'), ...stableOptions }
	// where stable options are spread after and take precedence.
	if unstable, ok := config["unstable"].(map[string]any); ok {
		setRawFieldsFromConfig(&p, infos, unstable)
	}

	// Process path-based config (VS Code style nested paths).
	// These run after unstable, so stable config values take precedence.
	for _, info := range infos {
		if info.configPath == "" {
			continue
		}
		configPath := configPathInfo{path: info.configPath, invert: info.configInvert}
		val, ok := getNestedValue(config, configPath.path)
		if !ok {
			for _, fallbackConfigPath := range info.fallbackConfigPaths {
				val, ok = getNestedValue(config, fallbackConfigPath.path)
				if ok {
					configPath = fallbackConfigPath
					break
				}
			}
		}
		if !ok {
			continue
		}

		if configPath.invert {
			if b, ok := val.(bool); ok {
				val = !b
			}
		}
		if parser, ok := configPathParsers[configPath.path]; ok {
			val = parser(val)
		}
		info.set(&p, val)
	}

	// Validate CustomConfigFileName for path traversal
	if p.CustomConfigFileName != "" {
		name := strings.TrimSpace(p.CustomConfigFileName)
		if strings.ContainsAny(name, "/\\") || name == ".." || name == "." {
			p.CustomConfigFileName = ""
		} else {
			p.CustomConfigFileName = name
		}
	}

	return p
}

func (p *UserPreferences) MarshalJSONTo(enc *json.Encoder) error {
	config := make(map[string]any)

	for _, info := range fieldInfoCache() {
		val := info.serialize(p)
		if val == nil {
			continue
		}

		// Prefer config path if available, otherwise use unstable section
		if info.configPath != "" {
			if info.configInvert {
				if b, ok := val.(bool); ok {
					val = !b
				}
			}
			setNestedValue(config, info.configPath, val)
		} else if info.rawName != "" {
			if info.rawInvert {
				if b, ok := val.(bool); ok {
					val = !b
				}
			}
			setNestedValue(config, "unstable."+info.rawName, val)
		}
	}

	return json.MarshalEncode(enc, config, json.Deterministic(true))
}

func (p *UserPreferences) UnmarshalJSONFrom(dec *json.Decoder) error {
	var config map[string]any
	if err := json.UnmarshalDecode(dec, &config); err != nil {
		return err
	}
	// Start with defaults, then overlay parsed values
	*p = NewDefaultUserPreferences().withConfig(config)
	return nil
}

// --- Helper methods ---

// WithOverrides returns a copy of p with non-zero fields from overrides applied on top.
// This is safe because all preference fields use types where zero = "not set":
// Tristate (TSUnknown=0), int (0), string (""), slice (nil).
func (p UserPreferences) WithOverrides(overrides UserPreferences) UserPreferences {
	for _, info := range fieldInfoCache() {
		info.merge(&p, overrides)
	}
	return p
}

func (p UserPreferences) ModuleSpecifierPreferences() modulespecifiers.UserPreferences {
	return modulespecifiers.UserPreferences{
		ImportModuleSpecifierPreference:   p.ImportModuleSpecifierPreference,
		ImportModuleSpecifierEnding:       p.ImportModuleSpecifierEnding,
		AutoImportSpecifierExcludeRegexes: p.AutoImportSpecifierExcludeRegexes,
	}
}

func (p UserPreferences) ParsedAutoImportFileExcludePatterns(useCaseSensitiveFileNames bool) *vfsmatch.SpecMatcher {
	return vfsmatch.NewSpecMatcher(p.AutoImportFileExcludePatterns, "", vfsmatch.UsageExclude, useCaseSensitiveFileNames)
}

func (p UserPreferences) IsModuleSpecifierExcluded(moduleSpecifier string) bool {
	return modulespecifiers.IsExcludedByRegex(moduleSpecifier, p.AutoImportSpecifierExcludeRegexes)
}

func ParseUserPreferences(items map[string]any) UserPreferences {
	prefs := NewDefaultUserPreferences()
	// Apply editor settings first (tabSize, indentSize, etc.) as raw-name defaults,
	// then overlay language-specific settings with increasing precedence:
	// editor < javascript < typescript < js/ts
	if editorItem, ok := items["editor"]; ok && editorItem != nil {
		if editorSettings, ok := editorItem.(map[string]any); ok {
			normalizedSettings := make(map[string]any, len(editorSettings)+2)
			maps.Copy(normalizedSettings, editorSettings)
			if tabSize, ok := normalizedSettings["tabSize"]; ok {
				if _, hasIndentSize := normalizedSettings["indentSize"]; !hasIndentSize {
					normalizedSettings["indentSize"] = tabSize
				}
			}
			if insertSpaces, ok := normalizedSettings["insertSpaces"]; ok {
				if _, hasConvertTabsToSpaces := normalizedSettings["convertTabsToSpaces"]; !hasConvertTabsToSpaces {
					normalizedSettings["convertTabsToSpaces"] = insertSpaces
				}
			}
			prefs = prefs.withConfig(map[string]any{"unstable": normalizedSettings})
		}
	}
	// Apply javascript, then typescript, then js/ts (highest precedence).
	for _, section := range []string{"javascript", "typescript", "js/ts"} {
		if item, ok := items[section]; ok && item != nil {
			if settings, ok := item.(map[string]any); ok {
				prefs = prefs.withConfig(settings)
			}
		}
	}
	return prefs
}
