//// [tests/cases/conformance/parser/ecmascript5/ErrorRecovery/Blocks/parserErrorRecovery_Block2.ts] ////

//// [parserErrorRecovery_Block2.ts]
function f() {
    ¬
    return;
}

//// [parserErrorRecovery_Block2.js]
"use strict";
function f() {
    return;
}
