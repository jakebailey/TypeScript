//// [tests/cases/compiler/ambientModules.ts] ////

//// [ambientModules.ts]
declare module Foo.Bar { export var foo; };
Foo.Bar.foo = 5; 

//// [ambientModules.js]
"use strict";
;
Foo.Bar.foo = 5;
