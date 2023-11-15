//// [tests/cases/compiler/APILibCheck_typescript-internal.ts] ////

//// [package.json]
{
    "name": "typescript",
    "types": "/.ts/typescript.internal.d.ts"
}

//// [index.ts]
import ts = require("typescript");


//// [index.js]
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
