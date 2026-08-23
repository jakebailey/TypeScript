//// [tests/cases/compiler/parseNotEqualsEqualsEquals.ts] ////

//// [parseNotEqualsEqualsEquals.ts]
declare let a: number, b: number;

a !=== b;

a == b;
a != b;
a === b;
a !== b;

a !=== b;
a !=== b;


//// [parseNotEqualsEqualsEquals.js]
"use strict";
a !== b;
a == b;
a != b;
a === b;
a !== b;
a !== b;
a !== b;
