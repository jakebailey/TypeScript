//// [tests/cases/conformance/es6/destructuring/destructuringTypeAssertionsES5_3.ts] ////

//// [destructuringTypeAssertionsES5_3.ts]
var { x } = <any>(foo());

//// [destructuringTypeAssertionsES5_3.js]
"use strict";
var x = (foo()).x;
