//// [tests/cases/compiler/bind1.ts] ////

//// [bind1.ts]
module M {
    export class C implements I {} // this should be an unresolved symbol I error
}



//// [bind1.js]
"use strict";
var M;
(function (M) {
    var C = /** @class */ (function () {
        function C() {
        }
        return C;
    }()); // this should be an unresolved symbol I error
    M.C = C;
})(M || (M = {}));
