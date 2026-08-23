//// [tests/cases/compiler/narrowingDisjointLiteralUnions.ts] ////

//// [narrowingDisjointLiteralUnions.ts]
type StringsA = "a" | "b";
type StringsB = "c" | "d";
declare function isStringsB(value: string): value is StringsB;

function narrowStrings(value: StringsA | StringsB) {
    if (isStringsB(value)) {
        const narrowed: StringsB = value;
    }
    else {
        const narrowed: StringsA = value;
    }
}

type NumbersA = 0 | 1;
type NumbersB = 2 | 3;
declare function isNumbersB(value: number): value is NumbersB;

function narrowNumbers(value: NumbersA | NumbersB) {
    if (isNumbersB(value)) {
        const narrowed: NumbersB = value;
    }
    else {
        const narrowed: NumbersA = value;
    }
}

type BigIntsA = 0n | 1n;
type BigIntsB = 2n | 3n;
declare function isBigIntsB(value: bigint): value is BigIntsB;

function narrowBigInts(value: BigIntsA | BigIntsB) {
    if (isBigIntsB(value)) {
        const narrowed: BigIntsB = value;
    }
    else {
        const narrowed: BigIntsA = value;
    }
}

declare function isTrue(value: boolean): value is true;

function narrowBooleans(value: boolean) {
    if (isTrue(value)) {
        const narrowed: true = value;
    }
    else {
        const narrowed: false = value;
    }
}

enum A {
    Value = "same",
}

enum B {
    Value = "same",
}

declare function isB(value: A | B): value is B;

function preserveEnumRelations(value: A | B) {
    if (isB(value)) {
        const narrowed: B = value;
    }
    else {
        const narrowed: A = value;
    }
}


//// [narrowingDisjointLiteralUnions.js]
"use strict";
function narrowStrings(value) {
    if (isStringsB(value)) {
        const narrowed = value;
    }
    else {
        const narrowed = value;
    }
}
function narrowNumbers(value) {
    if (isNumbersB(value)) {
        const narrowed = value;
    }
    else {
        const narrowed = value;
    }
}
function narrowBigInts(value) {
    if (isBigIntsB(value)) {
        const narrowed = value;
    }
    else {
        const narrowed = value;
    }
}
function narrowBooleans(value) {
    if (isTrue(value)) {
        const narrowed = value;
    }
    else {
        const narrowed = value;
    }
}
var A;
(function (A) {
    A["Value"] = "same";
})(A || (A = {}));
var B;
(function (B) {
    B["Value"] = "same";
})(B || (B = {}));
function preserveEnumRelations(value) {
    if (isB(value)) {
        const narrowed = value;
    }
    else {
        const narrowed = value;
    }
}
