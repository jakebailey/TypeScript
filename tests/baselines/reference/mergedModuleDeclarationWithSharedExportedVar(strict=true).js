//// [tests/cases/compiler/mergedModuleDeclarationWithSharedExportedVar.ts] ////

//// [mergedModuleDeclarationWithSharedExportedVar.ts]
module M {
    export var v = 10;
    v;
}
module M {
    v;
}

//// [mergedModuleDeclarationWithSharedExportedVar.js]
"use strict";
var M;
(function (M) {
    M.v = 10;
    M.v;
})(M || (M = {}));
(function (M) {
    M.v;
})(M || (M = {}));
