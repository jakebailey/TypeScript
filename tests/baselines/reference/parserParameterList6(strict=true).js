//// [tests/cases/conformance/parser/ecmascript5/ParameterLists/parserParameterList6.ts] ////

//// [parserParameterList6.ts]
class C {
  constructor(C: (public A) => any) {
  }
}

//// [parserParameterList6.js]
"use strict";
var C = /** @class */ (function () {
    function C(C) {
    }
    return C;
}());
