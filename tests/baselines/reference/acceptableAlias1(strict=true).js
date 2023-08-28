//// [tests/cases/compiler/acceptableAlias1.ts] ////

//// [acceptableAlias1.ts]
module M {
    export module N {
    }
    export import X = N;
}

import r = M.X;

//// [acceptableAlias1.js]
"use strict";
var M;
(function (M) {
})(M || (M = {}));
