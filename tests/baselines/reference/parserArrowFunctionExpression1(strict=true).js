//// [tests/cases/conformance/parser/ecmascript5/ArrowFunctionExpressions/parserArrowFunctionExpression1.ts] ////

//// [parserArrowFunctionExpression1.ts]
var v = (public x: string) => { };

//// [parserArrowFunctionExpression1.js]
"use strict";
var v = function (x) { };
