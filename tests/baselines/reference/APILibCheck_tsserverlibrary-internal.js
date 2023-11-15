//// [tests/cases/compiler/APILibCheck_tsserverlibrary-internal.ts] ////

//// [package.json]
{
    "name": "typescript",
    "types": "/.ts/tsserverlibrary.internal.d.ts"
}

//// [index.ts]
import ts = require("typescript");


//// [index.js]
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
