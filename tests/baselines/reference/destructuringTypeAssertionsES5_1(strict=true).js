//// [tests/cases/conformance/es6/destructuring/destructuringTypeAssertionsES5_1.ts] ////

//// [destructuringTypeAssertionsES5_1.ts]
var { x } = <any>foo();

//// [destructuringTypeAssertionsES5_1.js]
"use strict";
var x = foo().x;
