/// <reference path="../fourslash.ts" />
// @Filename: /home/src/workspaces/project/folder/target.ts
////[||]

// @Filename: /home/src/workspaces/project/a.ts
//// function foo() { }
//// const x = foo()
////
//// // comment
//// [|foo
//// /* another comment */|]
//// x

// @Filename: /home/src/workspaces/project/tsconfig.json
////{ "files": ["a.ts", "folder/target.ts"] }

const ranges = test.ranges();
verify.pasteEdits({
    args: {
        pastedText: [`foo
/* another comment */`],
        pasteLocations: [ranges[0]],
        copiedFrom: { file: "/home/src/workspaces/project/a.ts", range: [ranges[1]] },
    },
    newFileContents: {
        "/home/src/workspaces/project/folder/target.ts":
`import { foo } from "../a";

foo
/* another comment */`,
        "/home/src/workspaces/project/a.ts":
`export function foo() { }
const x = foo()

// comment
foo
/* another comment */
x`,
    }
});
