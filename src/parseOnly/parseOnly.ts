import {
    createSourceFile,
} from "./compiler/parser";
import {
    ScriptTarget,
} from "./compiler/types";

const iterations = 100;

declare const CHECKER_TS: string;

const source = CHECKER_TS;

for (let i = 0; i < iterations; i++) {
    createSourceFile("checker.ts", source, ScriptTarget.Latest);
}
