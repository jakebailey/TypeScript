//// [tests/cases/conformance/parser/ecmascript5/IndexMemberDeclarations/parserIndexMemberDeclaration9.ts] ////

//// [parserIndexMemberDeclaration9.ts]
class C {
   export [x: string]: string;
}

//// [parserIndexMemberDeclaration9.js]
"use strict";
var C = /** @class */ (function () {
    function C() {
    }
    return C;
}());
