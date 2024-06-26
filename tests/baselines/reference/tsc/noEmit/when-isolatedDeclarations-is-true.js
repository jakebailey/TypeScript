currentDirectory:: / useCaseSensitiveFileNames: false
Input::
//// [/lib/lib.d.ts]
/// <reference no-default-lib="true"/>
interface Boolean {}
interface Function {}
interface CallableFunction {}
interface NewableFunction {}
interface IArguments {}
interface Number { toExponential: any; }
interface Object {}
interface RegExp {}
interface String { charAt: any; }
interface Array<T> { length: number; [n: number]: T; }
interface ReadonlyArray<T> {}
declare const console: { log(msg: any): void; };

//// [/src/project/class1.ts]
function errorOnAssignmentBelowDecl(): void {}
errorOnAssignmentBelowDecl.a = "";


//// [/src/project/tsconfig.json]
{
  "compilerOptions": {
    "declaration": true,
    "isolatedDeclarations": true
  }
}



Output::
/lib/tsc -noEmit -p src/project
exitCode:: ExitStatus.Success
Program root files: [
  "/src/project/class1.ts"
]
Program options: {
  "declaration": true,
  "isolatedDeclarations": true,
  "noEmit": true,
  "project": "/src/project",
  "configFilePath": "/src/project/tsconfig.json"
}
Program structureReused: Not
Program files::
/lib/lib.d.ts
/src/project/class1.ts




Change:: no-change-run
Input::


Output::
/lib/tsc -noEmit -p src/project
exitCode:: ExitStatus.Success
Program root files: [
  "/src/project/class1.ts"
]
Program options: {
  "declaration": true,
  "isolatedDeclarations": true,
  "noEmit": true,
  "project": "/src/project",
  "configFilePath": "/src/project/tsconfig.json"
}
Program structureReused: Not
Program files::
/lib/lib.d.ts
/src/project/class1.ts


