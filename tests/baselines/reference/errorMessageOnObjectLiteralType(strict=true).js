//// [tests/cases/compiler/errorMessageOnObjectLiteralType.ts] ////

//// [errorMessageOnObjectLiteralType.ts]
var x: {
    a: string;
    b: number;
};
x.getOwnPropertyNamess();
Object.getOwnPropertyNamess(null);

//// [errorMessageOnObjectLiteralType.js]
"use strict";
var x;
x.getOwnPropertyNamess();
Object.getOwnPropertyNamess(null);
