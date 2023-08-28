//// [tests/cases/compiler/interfaceNameAsIdentifier.ts] ////

//// [interfaceNameAsIdentifier.ts]
interface C {
    (): void;
}
C();

module m2 {
    export interface C {
        (): void;
    }
}

m2.C();


//// [interfaceNameAsIdentifier.js]
"use strict";
C();
m2.C();
