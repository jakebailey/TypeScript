//// [tests/cases/compiler/resolveInterfaceNameWithSameLetDeclarationName2.ts] ////

//// [resolveInterfaceNameWithSameLetDeclarationName2.ts]
interface foo { }
interface bar { }
let bar: bar | foo;
let foo: bar | foo;

//// [resolveInterfaceNameWithSameLetDeclarationName2.js]
"use strict";
var bar;
var foo;
