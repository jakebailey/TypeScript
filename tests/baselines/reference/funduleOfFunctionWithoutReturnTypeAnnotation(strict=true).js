//// [tests/cases/compiler/funduleOfFunctionWithoutReturnTypeAnnotation.ts] ////

//// [funduleOfFunctionWithoutReturnTypeAnnotation.ts]
function fn() {
    return fn.n;
}
module fn {
    export var n = 1;
}


//// [funduleOfFunctionWithoutReturnTypeAnnotation.js]
"use strict";
function fn() {
    return fn.n;
}
(function (fn) {
    fn.n = 1;
})(fn || (fn = {}));
