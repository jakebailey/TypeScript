import * as fc from "fast-check";

import * as ts from "../_namespaces/ts";

describe("unittests:: fuzz", () => {
    if (!process.env.ENABLE_FUZZING) {
        return;
    }

    it("createSourceFile", () => {
        fc.assert(
            fc.property(
                fc.constantFrom(...ts.supportedTSExtensionsFlat),
                fc.string(),
                fc.integer({ min: ts.ScriptTarget.ES3, max: ts.ScriptTarget.Latest }),
                fc.boolean(),
                fc.integer({ min: ts.ScriptKind.Unknown, max: ts.ScriptKind.Deferred }),
                (ext, sourceText, scriptTarget, setParentNodes, scriptKind) => {
                    ts.createSourceFile(`index${ext}`, sourceText, scriptTarget, setParentNodes, scriptKind);
                },
            ),
            { numRuns: 1_000_000 },
        );
    });
});
