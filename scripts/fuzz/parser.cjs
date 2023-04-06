const { FuzzedDataProvider } = require("@jazzer.js/core");


/** @type {import("../../built/local/typescript.internal")} */
const ts = /** @type {any} */ (require("../../built/local/typescript"));

const extensions = [...ts.supportedTSExtensionsFlat, ...ts.supportedJSExtensionsFlat, ts.Extension.Json];

/**
 * @param { Buffer } fuzzerInputData
 */
module.exports.fuzz = function fuzz(fuzzerInputData) {
    const data = new FuzzedDataProvider(fuzzerInputData);

    const ext = data.pickValue(extensions);
    const scriptTarget = data.consumeFloatInRange(ts.ScriptTarget.ES3, ts.ScriptTarget.Latest);
    const setParentNodes = data.consumeBoolean();
    const sourceText = data.consumeString(4 * 1024 * 1024, "utf-8", /*printable*/ true); // see editorServices.ts

    const printCaseFirst = !!process.env.PRINT_CASE;
    function printCase() {
        console.log(JSON.stringify({
            ext,
            sourceText,
            scriptTarget: ts.ScriptTarget[scriptTarget],
            setParentNodes,
        }, undefined, 4));
    }

    if (printCaseFirst) {
        printCase();
    }

    let crashed = true;
    try {
        ts.createSourceFile(`index${ext}`, sourceText, scriptTarget, setParentNodes);
        crashed = false;
    }
    finally {
        if (crashed && !printCaseFirst) {
            printCase();
        }
    }
};
