//// [tests/cases/compiler/declarationEmitNoTypeParamLeak1.ts] ////

//// [declarationEmitNoTypeParamLeak1.ts]
type BrokenType<T> = 'a' | 'b';

class MyClass {
    constructor(readonly arg?: BrokenType<any>, prop = [null as any as BrokenType<any>][0]) {}
}

//// [declarationEmitNoTypeParamLeak1.js]
"use strict";
var MyClass = /** @class */ (function () {
    function MyClass(arg, prop) {
        if (prop === void 0) { prop = [null][0]; }
        this.arg = arg;
    }
    return MyClass;
}());


//// [declarationEmitNoTypeParamLeak1.d.ts]
type BrokenType<T> = 'a' | 'b';
declare class MyClass {
    readonly arg?: BrokenType<any> | undefined;
    constructor(arg?: BrokenType<any> | undefined, prop?: BrokenType<T>);
}


//// [DtsFileErrors]


declarationEmitNoTypeParamLeak1.d.ts(4,70): error TS2304: Cannot find name 'T'.


==== declarationEmitNoTypeParamLeak1.d.ts (1 errors) ====
    type BrokenType<T> = 'a' | 'b';
    declare class MyClass {
        readonly arg?: BrokenType<any> | undefined;
        constructor(arg?: BrokenType<any> | undefined, prop?: BrokenType<T>);
                                                                         ~
!!! error TS2304: Cannot find name 'T'.
    }
    