//// [tests/cases/conformance/parser/ecmascript5/ArrowFunctionExpressions/parserArrowFunctionExpression3.ts] ////

//// [parserArrowFunctionExpression3.ts]
a = (() => { } || a)

//// [parserArrowFunctionExpression3.js]
"use strict";
a = (function () { }) || a;
