//// [tests/cases/compiler/APILibCheck_tsserverlibrary.ts] ////

//// [package.json]
{
    "name": "typescript",
    "types": "/.ts/tsserverlibrary.d.ts"
}

//// [index.ts]
import ts = require("typescript");


//// [index.js]
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
