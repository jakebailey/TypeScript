//// [tests/cases/compiler/forInStatement2.ts] ////

//// [forInStatement2.ts]
var expr: number;
for (var a in expr) {
}

//// [forInStatement2.js]
"use strict";
var expr;
for (var a in expr) {
}
