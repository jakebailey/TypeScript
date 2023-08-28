//// [tests/cases/conformance/es6/for-ofStatements/for-of27.ts] ////

//// [for-of27.ts]
class StringIterator {
    [Symbol.iterator]: any;
}

for (var v of new StringIterator) { }

//// [for-of27.js]
"use strict";
class StringIterator {
}
Symbol.iterator;
for (var v of new StringIterator) { }
