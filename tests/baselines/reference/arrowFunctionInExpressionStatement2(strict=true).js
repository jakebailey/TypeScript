//// [tests/cases/compiler/arrowFunctionInExpressionStatement2.ts] ////

//// [arrowFunctionInExpressionStatement2.ts]
module M {
    () => 0;
}

//// [arrowFunctionInExpressionStatement2.js]
"use strict";
var M;
(function (M) {
    (function () { return 0; });
})(M || (M = {}));
