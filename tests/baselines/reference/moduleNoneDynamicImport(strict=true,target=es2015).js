//// [tests/cases/compiler/moduleNoneDynamicImport.ts] ////

//// [a.ts]
const foo = import("./b");

//// [b.js]
export default 1;


//// [a.js]
"use strict";
const foo = Promise.resolve().then(() => require("b"));
