//// [tests/cases/conformance/es6/Symbols/symbolProperty17.ts] ////

//// [symbolProperty17.ts]
interface I {
    [Symbol.iterator]: number;
    [s: symbol]: string;
    "__@iterator": string;
}

var i: I;
var it = i[Symbol.iterator];

//// [symbolProperty17.js]
"use strict";
var i;
var it = i[Symbol.iterator];
