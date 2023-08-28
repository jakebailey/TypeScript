//// [tests/cases/conformance/es6/unicodeExtendedEscapes/unicodeExtendedEscapesInTemplates12_ES5.ts] ////

//// [unicodeExtendedEscapesInTemplates12_ES5.ts]
var x = `\u{FFFFFFFF}`;


//// [unicodeExtendedEscapesInTemplates12_ES5.js]
"use strict";
var x = "\\u{FFFFFFFF}";
