const { FuzzedDataProvider } = require("@jazzer.js/core");


/** @type {import("../../built/local/typescript.internal")} */
const ts = /** @type {any} */ (require("../../built/local/typescript"));

/**
 * @param { Buffer } fuzzerInputData
 */
module.exports.fuzz = function fuzz(fuzzerInputData) {
    const data = new FuzzedDataProvider(fuzzerInputData);

    const reportDiagnostics = data.consumeBoolean();
    const sourceText = data.consumeString(4 * 1024 * 1024, "utf-8", /*printable*/ true); // see editorServices.ts

    if (process.env.PRINT_CASE) {
        console.log(JSON.stringify({
            sourceText,
            reportDiagnostics,
        }, undefined, 4));
    }

    ts.transpileModule(sourceText, {
        reportDiagnostics,
    });
};
