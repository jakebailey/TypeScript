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
