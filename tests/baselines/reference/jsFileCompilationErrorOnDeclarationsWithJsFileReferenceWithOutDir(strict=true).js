//// [tests/cases/compiler/jsFileCompilationErrorOnDeclarationsWithJsFileReferenceWithOutDir.ts] ////

//// [a.ts]
class c {
}

//// [b.ts]
/// <reference path="c.js"/>
function foo() {
}

//// [c.js]
function bar() {
}

//// [a.js]
"use strict";
var c = /** @class */ (function () {
    function c() {
    }
    return c;
}());
//// [c.js]
"use strict";
function bar() {
}
//// [b.js]
"use strict";
/// <reference path="c.js"/>
function foo() {
}


//// [a.d.ts]
declare class c {
}
//// [c.d.ts]
declare function bar(): void;
//// [b.d.ts]
declare function foo(): void;
