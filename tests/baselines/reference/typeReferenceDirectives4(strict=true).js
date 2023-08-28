//// [tests/cases/compiler/typeReferenceDirectives4.ts] ////

//// [ref.d.ts]
interface $ { x }

//// [index.d.ts]
declare let $: { x: number }


//// [app.ts]
/// <reference path="./ref.d.ts"/>
/// <reference types="lib" preserve="true" />

let x: $;
let y = () => x

//// [app.js]
"use strict";
/// <reference path="./ref.d.ts"/>
/// <reference types="lib" preserve="true" />
var x;
var y = function () { return x; };


//// [app.d.ts]
/// <reference types="lib" preserve="true" />
declare let x: $;
declare let y: () => $;
