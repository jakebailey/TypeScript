//// [tests/cases/conformance/parser/ecmascript5/MemberVariableDeclarations/parserMemberVariableDeclaration3.ts] ////

//// [parserMemberVariableDeclaration3.ts]
class C {
  static public Foo;
}

//// [parserMemberVariableDeclaration3.js]
"use strict";
var C = /** @class */ (function () {
    function C() {
    }
    return C;
}());
