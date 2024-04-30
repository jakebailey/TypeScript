//// [tests/cases/compiler/APILibCheck.ts] ////

//// [package.json]
{
    "name": "typescript",
    "type": "module",
    "exports": "./lib/typescript.d.ts"
}

//// [package.json]
{
    "name": "typescript-internal",
    "type": "module",
    "exports": "./lib/typescript.internal.d.ts"
}

//// [package.json]
{
    "name": "tsserverlibrary",
    "type": "module",
    "exports": "./lib/tsserverlibrary.d.ts"
}

//// [package.json]
{
    "name": "tsserverlibrary-internal",
    "type": "module",
    "exports": "./lib/tsserverlibrary.internal.d.ts"
}

//// [package.json]
{
    "name": "project",
    "type": "module"
}

//// [index.ts]
import * as ts from "typescript";
import tsDefault from "typescript";
import * as tsInternal from "typescript-internal";
import tsInternalDefault from "typescript-internal";
import * as tsserverlibrary from "tsserverlibrary";
import tsserverlibraryDefault from "tsserverlibrary";
import * as tsserverlibraryInternal from "tsserverlibrary-internal";
import tsserverlibraryInternalDefault from "tsserverlibrary-internal";


//// [index.js]
export {};
