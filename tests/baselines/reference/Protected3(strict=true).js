//// [tests/cases/conformance/parser/ecmascript5/Protected/Protected3.ts] ////

//// [Protected3.ts]
class C {
  protected constructor() { }
}

//// [Protected3.js]
"use strict";
var C = /** @class */ (function () {
    function C() {
    }
    return C;
}());
