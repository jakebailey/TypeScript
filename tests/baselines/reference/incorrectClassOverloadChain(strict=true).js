//// [tests/cases/compiler/incorrectClassOverloadChain.ts] ////

//// [incorrectClassOverloadChain.ts]
class C {
    foo(): string;
    foo(x): number;
    x = 1;
}

//// [incorrectClassOverloadChain.js]
"use strict";
var C = /** @class */ (function () {
    function C() {
        this.x = 1;
    }
    return C;
}());
