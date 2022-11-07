import * as ts from "../../_namespaces/ts";
import * as fakes from "../../_namespaces/fakes";

describe("unittests:: tsbuild - output file paths", () => {
    const noChangeProject: ts.TestTscEdit = {
        modifyFs: ts.noop,
        subScenario: "Normal build without change, that does not block emit on error to show files that get emitted",
        commandLineArgs: ["-p", "/src/tsconfig.json"],
    };
    const edits: ts.TestTscEdit[] = [
        ts.noChangeRun,
        noChangeProject,
    ];

    function verify(input: Pick<ts.VerifyTscWithEditsInput, "subScenario" | "fs" | "edits">, expectedOuptutNames: readonly string[]) {
        ts.verifyTscWithEdits({
            scenario: "outputPaths",
            commandLineArgs: ["--b", "/src/tsconfig.json", "-v"],
            ...input
        });

        it("verify getOutputFileNames", () => {
            const sys = new fakes.System(input.fs().makeReadonly(), { executingFilePath: "/lib/tsc" }) as ts.TscCompileSystem;

            assert.deepEqual(
                ts.getOutputFileNames(
                    ts.parseConfigFileWithSystem("/src/tsconfig.json", {}, /*extendedConfigCache*/ undefined, {}, sys, ts.noop)!,
                    "/src/src/index.ts",
                    /*ignoreCase*/ false
                ),
                expectedOuptutNames
            );
        });
    }

    verify({
        subScenario: "when rootDir is not specified",
        fs: () => ts.loadProjectFromFiles({
            "/src/src/index.ts": "export const x = 10;",
            "/src/tsconfig.json": JSON.stringify({
                compilerOptions: {
                    outDir: "dist"
                }
            })
        }),
        edits,
    }, ["/src/dist/index.js"]);

    verify({
        subScenario: "when rootDir is not specified and is composite",
        fs: () => ts.loadProjectFromFiles({
            "/src/src/index.ts": "export const x = 10;",
            "/src/tsconfig.json": JSON.stringify({
                compilerOptions: {
                    outDir: "dist",
                    composite: true
                }
            })
        }),
        edits,
    }, ["/src/dist/src/index.js", "/src/dist/src/index.d.ts"]);

    verify({
        subScenario: "when rootDir is specified",
        fs: () => ts.loadProjectFromFiles({
            "/src/src/index.ts": "export const x = 10;",
            "/src/tsconfig.json": JSON.stringify({
                compilerOptions: {
                    outDir: "dist",
                    rootDir: "src"
                }
            })
        }),
        edits,
    }, ["/src/dist/index.js"]);

    verify({
        subScenario: "when rootDir is specified but not all files belong to rootDir",
        fs: () => ts.loadProjectFromFiles({
            "/src/src/index.ts": "export const x = 10;",
            "/src/types/type.ts": "export type t = string;",
            "/src/tsconfig.json": JSON.stringify({
                compilerOptions: {
                    outDir: "dist",
                    rootDir: "src"
                }
            })
        }),
        edits,
    }, ["/src/dist/index.js"]);

    verify({
        subScenario: "when rootDir is specified but not all files belong to rootDir and is composite",
        fs: () => ts.loadProjectFromFiles({
            "/src/src/index.ts": "export const x = 10;",
            "/src/types/type.ts": "export type t = string;",
            "/src/tsconfig.json": JSON.stringify({
                compilerOptions: {
                    outDir: "dist",
                    rootDir: "src",
                    composite: true
                }
            })
        }),
        edits,
    }, ["/src/dist/index.js", "/src/dist/index.d.ts"]);
});
