//// [tests/cases/conformance/jsdoc/declarations/jsDeclarationsRestArgsWithThisTypeInJSDocFunction.ts] ////

//// [bug38550.js]
export class Clazz {
  /**
   * @param {function(this:Object, ...*):*} functionDeclaration
   */
  method(functionDeclaration) {}
}


//// [bug38550.js]
export class Clazz {
    /**
     * @param {function(this:Object, ...*):*} functionDeclaration
     */
    method(functionDeclaration) { }
}


//// [bug38550.d.ts]
export declare class Clazz {
    /**
     * @param {function(this:Object, ...*):*} functionDeclaration
     */
    method(functionDeclaration: (this: Object, : any[]) => any): void;
}


//// [DtsFileErrors]


out/bug38550.d.ts(5,48): error TS1138: Parameter declaration expected.
out/bug38550.d.ts(5,50): error TS7051: Parameter has a name but no type. Did you mean 'arg1: any'?
out/bug38550.d.ts(5,53): error TS1005: ',' expected.


==== out/bug38550.d.ts (3 errors) ====
    export declare class Clazz {
        /**
         * @param {function(this:Object, ...*):*} functionDeclaration
         */
        method(functionDeclaration: (this: Object, : any[]) => any): void;
                                                   ~
!!! error TS1138: Parameter declaration expected.
                                                     ~~~
!!! error TS7051: Parameter has a name but no type. Did you mean 'arg1: any'?
                                                        ~
!!! error TS1005: ',' expected.
    }
    