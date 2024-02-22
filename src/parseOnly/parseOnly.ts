import {
    createSourceFile,
} from "./compiler/parser";
import {
    ScriptTarget,
} from "./compiler/types";

createSourceFile("test.ts", "const x = 10", ScriptTarget.Latest);
