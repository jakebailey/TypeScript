//// [tests/cases/compiler/defaultValueInFunctionOverload1.ts] ////

//// [defaultValueInFunctionOverload1.ts]
function foo(x: string = '');
function foo(x = '') { }

//// [defaultValueInFunctionOverload1.js]
"use strict";
function foo(x) {
    if (x === void 0) { x = ''; }
}
