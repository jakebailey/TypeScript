//// [tests/cases/compiler/declarationEmitNoTypeParamLeak1.ts] ////

//// [declarationEmitNoTypeParamLeak1.ts]
type BrokenType<T> = "a" | "b";

class MyClass {
    constructor(readonly arg?: BrokenType<any>, prop = [null as any as BrokenType<any>][0]) {}
}


//// [declarationEmitNoTypeParamLeak1.js]
"use strict";
class MyClass {
    arg;
    constructor(arg, prop = [null][0]) {
        this.arg = arg;
    }
}


//// [declarationEmitNoTypeParamLeak1.d.ts]
type BrokenType<T> = "a" | "b";
declare class MyClass {
    readonly arg?: BrokenType<any> | undefined;
    constructor(arg?: BrokenType<any> | undefined, prop?: BrokenType<any>);
}
