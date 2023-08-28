//// [tests/cases/compiler/commentOnElidedModule1.ts] ////

//// [a.ts]
/*!=================
    Keep this pinned
   =================
*/

/*! Don't keep this pinned comment */
module ElidedModule {
}

// Don't keep this comment.
module ElidedModule2 {
}

//// [b.ts]
///<reference path="a.ts"/>
module ElidedModule3 {
}

//// [a.js]
"use strict";
/*!=================
    Keep this pinned
   =================
*/
//// [b.js]
"use strict";
///<reference path="a.ts"/>
