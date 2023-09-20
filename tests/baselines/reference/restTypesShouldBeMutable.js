//// [tests/cases/compiler/restTypesShouldBeMutable.ts] ////

//// [restTypesShouldBeMutable.ts]
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


//// [restTypesShouldBeMutable.js]
"use strict";
callFn(input, function () {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    args;
    args[0] = "hello";
});
callFnNonGeneric(input, function () {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    args;
    args[0] = "hello";
});
callFnTuple(inputTuple, function () {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    args;
    args[0] = "hello";
});
callFnNonGenericTuple(inputTuple, function () {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
        args[_i] = arguments[_i];
    }
    args;
    args[0] = "hello";
});
