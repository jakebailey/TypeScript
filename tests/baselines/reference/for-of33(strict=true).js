//// [tests/cases/conformance/es6/for-ofStatements/for-of33.ts] ////

//// [for-of33.ts]
class StringIterator {
    [Symbol.iterator]() {
        return v;
    }
}

for (var v of new StringIterator) { }

//// [for-of33.js]
"use strict";
class StringIterator {
    [Symbol.iterator]() {
        return v;
    }
}
for (var v of new StringIterator) { }
