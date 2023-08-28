//// [tests/cases/compiler/propertyAccess2.ts] ////

//// [propertyAccess2.ts]
var foo: number;
foo.toBAZ();

//// [propertyAccess2.js]
"use strict";
var foo;
foo.toBAZ();
