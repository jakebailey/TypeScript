//// [tests/cases/compiler/modifierOnParameter1.ts] ////

//// [modifierOnParameter1.ts]
class C {
   constructor(declare p) { }
}

//// [modifierOnParameter1.js]
"use strict";
var C = /** @class */ (function () {
    function C(p) {
    }
    return C;
}());
