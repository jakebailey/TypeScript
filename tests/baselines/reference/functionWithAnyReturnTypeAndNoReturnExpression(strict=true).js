//// [tests/cases/compiler/functionWithAnyReturnTypeAndNoReturnExpression.ts] ////

//// [functionWithAnyReturnTypeAndNoReturnExpression.ts]
// All should be allowed
function f(): any { }
var f2: () => any = () => { };
var f3 = (): any => { };

//// [functionWithAnyReturnTypeAndNoReturnExpression.js]
"use strict";
// All should be allowed
function f() { }
var f2 = function () { };
var f3 = function () { };
