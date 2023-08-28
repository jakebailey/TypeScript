//// [tests/cases/compiler/internalImportUnInstantiatedModuleMergedWithClassNotReferencingInstanceNoConflict.ts] ////

//// [internalImportUnInstantiatedModuleMergedWithClassNotReferencingInstanceNoConflict.ts]
class A {
    aProp: string;
}
module A {
    export interface X { s: string }
}

module B {
    import Y = A;
}


//// [internalImportUnInstantiatedModuleMergedWithClassNotReferencingInstanceNoConflict.js]
"use strict";
var A = /** @class */ (function () {
    function A() {
    }
    return A;
}());
