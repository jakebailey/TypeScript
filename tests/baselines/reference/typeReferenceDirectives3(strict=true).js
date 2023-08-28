//// [tests/cases/compiler/typeReferenceDirectives3.ts] ////

//// [ref.d.ts]
interface $ { x }

//// [index.d.ts]
declare let $: { x: number }

//// [app.ts]
/// <reference types="lib" preserve="true" />
/// <reference path="ref.d.ts" />
interface A {
    x: () => $
}

//// [app.js]
"use strict";
/// <reference types="lib" preserve="true" />
/// <reference path="ref.d.ts" />


//// [app.d.ts]
/// <reference types="lib" preserve="true" />
interface A {
    x: () => $;
}
