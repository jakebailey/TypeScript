//// [tests/cases/compiler/moduleNoEmit.ts] ////

//// [moduleNoEmit.ts]
module Foo {
	1+1;
}

//// [moduleNoEmit.js]
"use strict";
var Foo;
(function (Foo) {
    1 + 1;
})(Foo || (Foo = {}));
