//// [tests/cases/compiler/propertyAccess1.ts] ////

//// [propertyAccess1.ts]
var foo: { a: number; };
foo.a = 4;
foo.b = 5;

//// [propertyAccess1.js]
"use strict";
var foo;
foo.a = 4;
foo.b = 5;
