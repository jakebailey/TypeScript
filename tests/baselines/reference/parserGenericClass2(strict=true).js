//// [tests/cases/conformance/parser/ecmascript5/Generics/parserGenericClass2.ts] ////

//// [parserGenericClass2.ts]
class C<K,V> {
}

//// [parserGenericClass2.js]
"use strict";
var C = /** @class */ (function () {
    function C() {
    }
    return C;
}());
