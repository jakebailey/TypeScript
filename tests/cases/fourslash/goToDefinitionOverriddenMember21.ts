/// <reference path="./fourslash.ts"/>

// @strict: true
// @target: esnext
// @lib: esnext

//// const prop = "foo" as const;
////
//// abstract class A {}
////
//// export class B extends A {
////   static [|/*1*/override|] readonly [prop] = "B";
//// }

verify.baselineGoToDefinition("1");
