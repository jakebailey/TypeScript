//// [tests/cases/conformance/es6/templates/templateStringControlCharacterEscapes02.ts] ////

//// [templateStringControlCharacterEscapes02.ts]
var x = `\x19\u0019 19`;

//// [templateStringControlCharacterEscapes02.js]
"use strict";
var x = "\u0019\u0019 19";
