import * as fc from "fast-check";

import * as ts from "../_namespaces/ts";

describe("unittests:: fuzz", () => {
    if (!process.env.ENABLE_FUZZING) {
        return;
    }

    it("createSourceFile", () => {
        fc.assert(
            fc.property(
                fc.constantFrom(...ts.supportedTSExtensionsFlat, ...ts.supportedJSExtensionsFlat, ts.Extension.Json),
                fc.string(),
                fc.integer({ min: ts.ScriptTarget.ES3, max: ts.ScriptTarget.Latest }),
                fc.boolean(),
                (ext, sourceText, scriptTarget, setParentNodes) => {
                    ts.createSourceFile(`index${ext}`, sourceText, scriptTarget, setParentNodes);
                },
            ),
            { numRuns: 1_000_000 },
        );
    });
});
