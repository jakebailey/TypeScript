//// [tests/cases/conformance/parser/ecmascript5/ErrorRecovery/AccessibilityAfterStatic/parserAccessibilityAfterStatic10.ts] ////

//// [parserAccessibilityAfterStatic10.ts]
class Outer
{
static public intI<T>() {}
}


//// [parserAccessibilityAfterStatic10.js]
"use strict";
var Outer = /** @class */ (function () {
    function Outer() {
    }
    Outer.intI = function () { };
    return Outer;
}());
