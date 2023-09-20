// @strict: true

declare function callFn<T extends readonly any[]>(args: T, fn: (...args: T) => void): void;

declare const input: readonly string[];

callFn(input, (...args) => {
    args;
    args[0] = "hello";
})

declare function callFnNonGeneric(args: readonly string[], fn: (...args: readonly string[]) => void): void;

callFnNonGeneric(input, (...args) => {
    args;
    args[0] = "hello";
})


declare function callFnTuple<T extends readonly [string, string, string]>(args: T, fn: (...args: T) => void): void;

declare const inputTuple: readonly [string, string, string];

callFnTuple(inputTuple, (...args) => {
    args;
    args[0] = "hello";
})

declare function callFnNonGenericTuple(args: readonly [string, string, string], fn: (...args: readonly [string, string, string]) => void): void;

callFnNonGenericTuple(inputTuple, (...args) => {
    args;
    args[0] = "hello";
})
