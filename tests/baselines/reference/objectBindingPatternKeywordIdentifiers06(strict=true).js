//// [tests/cases/conformance/es6/destructuring/objectBindingPatternKeywordIdentifiers06.ts] ////

//// [objectBindingPatternKeywordIdentifiers06.ts]
var { as: as } = { as: 1 }

//// [objectBindingPatternKeywordIdentifiers06.js]
"use strict";
var as = { as: 1 }.as;
