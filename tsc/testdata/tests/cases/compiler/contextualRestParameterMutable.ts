// @strict: true

declare function callFn<T extends readonly any[]>(args: T, fn: (...args: T) => void): void;
declare function callFnNonGeneric(args: readonly string[], fn: (...args: readonly string[]) => void): void;
declare function mapArgs<T extends readonly any[]>(args: T, fn: (...args: T) => T): T;

declare const input: readonly string[];

callFn(input, (...args) => {
    args.push("value");
    args[0] = "value";
});

callFnNonGeneric(input, (...args) => {
    args.push("value");
    args[0] = "value";
});

const mapped = mapArgs(input, (...args) => {
    args.push("value");
    return args;
});

mapped;
