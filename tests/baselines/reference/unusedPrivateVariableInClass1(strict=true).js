//// [tests/cases/compiler/unusedPrivateVariableInClass1.ts] ////

//// [unusedPrivateVariableInClass1.ts]
class greeter {
    private x: string;
}

//// [unusedPrivateVariableInClass1.js]
"use strict";
var greeter = /** @class */ (function () {
    function greeter() {
    }
    return greeter;
}());
