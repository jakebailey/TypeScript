//// [tests/cases/compiler/propertyAccess3.ts] ////

//// [propertyAccess3.ts]
var foo: boolean;
foo.toBAZ();

//// [propertyAccess3.js]
"use strict";
var foo;
foo.toBAZ();
