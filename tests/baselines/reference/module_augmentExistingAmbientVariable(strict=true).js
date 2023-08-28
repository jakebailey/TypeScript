//// [tests/cases/compiler/module_augmentExistingAmbientVariable.ts] ////

//// [module_augmentExistingAmbientVariable.ts]
declare var console: any;

module console {
    export var x = 2;
}

//// [module_augmentExistingAmbientVariable.js]
"use strict";
var console;
(function (console) {
    console.x = 2;
})(console || (console = {}));
