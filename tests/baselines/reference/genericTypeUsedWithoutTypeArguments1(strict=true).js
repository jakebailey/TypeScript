//// [tests/cases/compiler/genericTypeUsedWithoutTypeArguments1.ts] ////

//// [genericTypeUsedWithoutTypeArguments1.ts]
interface Foo<T> { }
class Bar<T> implements Foo { }


//// [genericTypeUsedWithoutTypeArguments1.js]
"use strict";
var Bar = /** @class */ (function () {
    function Bar() {
    }
    return Bar;
}());
