//// [tests/cases/conformance/types/literal/stringLiteralsWithSwitchStatements01.ts] ////

//// [stringLiteralsWithSwitchStatements01.ts]
let x: "foo";
let y: "foo" | "bar"; 

switch (x) {
    case "foo":
        break;
    case "bar":
        break;
    case y:
        y;
        break;
}


//// [stringLiteralsWithSwitchStatements01.js]
"use strict";
var x;
var y;
switch (x) {
    case "foo":
        break;
    case "bar":
        break;
    case y:
        y;
        break;
}
