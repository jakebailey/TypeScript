// @strict: true
// @declaration: true

type BrokenType<T> = 'a' | 'b';

class MyClass {
    constructor(readonly arg?: BrokenType<any>, prop = [null as any as BrokenType<any>][0]) {}
}