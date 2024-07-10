//// [tests/cases/compiler/issue58960.ts] ////

//// [issue58960.ts]
type A = any
// @ts-ignore
type B = DoesNotExist
type C = B & any

// @ts-expect-error
const a: 0 extends 1 & A ? true : false = false;
// @ts-expect-error
const b: 0 extends 1 & B ? true : false = false;
// @ts-expect-error
const c: 0 extends 1 & C ? true : false = false;


//// [issue58960.js]
"use strict";
// @ts-expect-error
var a = false;
// @ts-expect-error
var b = false;
// @ts-expect-error
var c = false;
