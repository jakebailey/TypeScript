//// [tests/cases/conformance/parser/ecmascript5/ConstructorDeclarations/parserConstructorDeclaration7.ts] ////

//// [parserConstructorDeclaration7.ts]
class C {
  public private constructor() { }
}

//// [parserConstructorDeclaration7.js]
"use strict";
var C = /** @class */ (function () {
    function C() {
    }
    return C;
}());
