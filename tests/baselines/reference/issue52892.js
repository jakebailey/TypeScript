//// [tests/cases/compiler/issue52892.ts] ////

//// [issue52892.ts]
class A<T = number> {
    //static { type _<T extends A = A> = 0 }

    value!: T;
    child!: InstanceType<typeof A.B<A<T>>>

    static B = class B<T extends A = A> {
        parent!: T;
    } 
}

var a = new A
a.child.parent.value


//// [issue52892.js]
"use strict";
var A = /** @class */ (function () {
    function A() {
    }
    A.B = /** @class */ (function () {
        function B() {
        }
        return B;
    }());
    return A;
}());
var a = new A;
a.child.parent.value;
