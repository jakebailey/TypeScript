//// [tests/cases/compiler/jsFileCompilationWithDeclarationEmitPathSameAsInput.ts] ////

//// [a.ts]
class c {
}

//// [a.d.ts]
declare function isC(): boolean;

//// [a.js]
"use strict";
var c = /** @class */ (function () {
    function c() {
    }
    return c;
}());
