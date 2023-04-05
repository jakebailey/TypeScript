const { FuzzedDataProvider } = require("@jazzer.js/core");


/** @type {import("../../built/local/typescript.internal")} */
const ts = /** @type {any} */ (require("../../built/local/typescript"));

const extensions = [...ts.supportedTSExtensionsFlat, ...ts.supportedJSExtensionsFlat, ts.Extension.Json];

/**
 * @param { Buffer } fuzzerInputData
 */
module.exports.fuzz = function fuzz(fuzzerInputData) {
    const data = new FuzzedDataProvider(fuzzerInputData);

    const ext = extensions[data.consumeIntegralInRange(0, extensions.length - 1)];
    const sourceText = data.consumeString(4 * 1024 * 1024); // see editorServices.ts
    const scriptTarget = data.consumeFloatInRange(ts.ScriptTarget.ES3, ts.ScriptTarget.Latest);
    const setParentNodes = data.consumeBoolean();

    ts.createSourceFile(`index${ext}`, sourceText, scriptTarget, setParentNodes);
};
