import * as ts from "../../_namespaces/ts.js";
import { dedent } from "../../_namespaces/Utils.js";
import { jsonToReadableText } from "../helpers.js";
import { compilerOptionsToConfigJson } from "../helpers/contents.js";
import {
    noChangeOnlyRuns,
    noChangeRun,
    TestTscEdit,
    verifyTsc,
} from "../helpers/tsc.js";
import {
    loadProjectFromFiles,
    replaceText,
} from "../helpers/vfs.js";
describe("unittests:: tsc:: noEmit::", () => {
    describe("when noEmit changes between compilation", () => {
        verifyNoEmitChanges({ incremental: true });
        verifyNoEmitChanges({ incremental: true, declaration: true });
        verifyNoEmitChanges({ composite: true });
        verifyNoEmitChanges({ incremental: true, outFile: "../outFile.js", module: ts.ModuleKind.AMD });
        verifyNoEmitChanges({ incremental: true, declaration: true, outFile: "../outFile.js", module: ts.ModuleKind.AMD });
        verifyNoEmitChanges({ composite: true, outFile: "../outFile.js", module: ts.ModuleKind.AMD });

        function verifyNoEmitChanges(compilerOptions: ts.CompilerOptions) {
            const discrepancyExplanation = () => [
                "Clean build will not have latestChangedDtsFile as there was no emit and emitSignatures as undefined for files",
                "Incremental will store the past latestChangedDtsFile and emitSignatures",
            ];
            const noChangeRunWithNoEmit: TestTscEdit = {
                ...noChangeRun,
                caption: "No Change run with noEmit",
                commandLineArgs: ["--p", "src/project", "--noEmit"],
                discrepancyExplanation: compilerOptions.composite ?
                    discrepancyExplanation :
                    undefined,
            };
            const noChangeRunWithEmit: TestTscEdit = {
                ...noChangeRun,
                caption: "No Change run with emit",
                commandLineArgs: ["--p", "src/project"],
            };
            let optionsString = "";
            for (const key in compilerOptions) {
                if (ts.hasProperty(compilerOptions, key) && key !== "outFile" && key !== "module") {
                    optionsString += ` ${key}`;
                }
            }
            function scenarioName(text: string) {
                return `${compilerOptions.outFile ? "outFile" : "multiFile"}/${text}${optionsString}`;
            }

            verifyTsc({
                scenario: "noEmit",
                subScenario: scenarioName("changes"),
                commandLineArgs: ["--p", "src/project"],
                fs,
                edits: [
                    noChangeRunWithNoEmit,
                    noChangeRunWithNoEmit,
                    {
                        caption: "Introduce error but still noEmit",
                        commandLineArgs: ["--p", "src/project", "--noEmit"],
                        edit: fs => replaceText(fs, "/src/project/src/class.ts", "prop", "prop1"),
                        discrepancyExplanation: compilerOptions.composite ?
                            discrepancyExplanation :
                            undefined,
                    },
                    {
                        caption: "Fix error and emit",
                        edit: fs => replaceText(fs, "/src/project/src/class.ts", "prop1", "prop"),
                    },
                    noChangeRunWithEmit,
                    noChangeRunWithNoEmit,
                    noChangeRunWithNoEmit,
                    noChangeRunWithEmit,
                    {
                        caption: "Introduce error and emit",
                        edit: fs => replaceText(fs, "/src/project/src/class.ts", "prop", "prop1"),
                    },
                    noChangeRunWithEmit,
                    noChangeRunWithNoEmit,
                    noChangeRunWithNoEmit,
                    noChangeRunWithEmit,
                    {
                        caption: "Fix error and no emit",
                        commandLineArgs: ["--p", "src/project", "--noEmit"],
                        edit: fs => replaceText(fs, "/src/project/src/class.ts", "prop1", "prop"),
                        discrepancyExplanation: compilerOptions.composite ?
                            discrepancyExplanation :
                            undefined,
                    },
                    noChangeRunWithEmit,
                    noChangeRunWithNoEmit,
                    noChangeRunWithNoEmit,
                    noChangeRunWithEmit,
                ],
            });

            verifyTsc({
                scenario: "noEmit",
                subScenario: scenarioName("changes with initial noEmit"),
                commandLineArgs: ["--p", "src/project", "--noEmit"],
                fs,
                edits: [
                    noChangeRunWithEmit,
                    {
                        caption: "Introduce error with emit",
                        commandLineArgs: ["--p", "src/project"],
                        edit: fs => replaceText(fs, "/src/project/src/class.ts", "prop", "prop1"),
                    },
                    {
                        caption: "Fix error and no emit",
                        edit: fs => replaceText(fs, "/src/project/src/class.ts", "prop1", "prop"),
                        discrepancyExplanation: compilerOptions.composite ?
                            discrepancyExplanation :
                            undefined,
                    },
                    noChangeRunWithEmit,
                ],
            });

            function fs() {
                return loadProjectFromFiles({
                    "/src/project/src/class.ts": dedent`
                            export class classC {
                                prop = 1;
                            }`,
                    "/src/project/src/indirectClass.ts": dedent`
                            import { classC } from './class';
                            export class indirectClass {
                                classC = new classC();
                            }`,
                    "/src/project/src/directUse.ts": dedent`
                            import { indirectClass } from './indirectClass';
                            new indirectClass().classC.prop;`,
                    "/src/project/src/indirectUse.ts": dedent`
                            import { indirectClass } from './indirectClass';
                            new indirectClass().classC.prop;`,
                    "/src/project/src/noChangeFile.ts": dedent`
                            export function writeLog(s: string) {
                            }`,
                    "/src/project/src/noChangeFileWithEmitSpecificError.ts": dedent`
                            function someFunc(arguments: boolean, ...rest: any[]) {
                            }`,
                    "/src/project/tsconfig.json": jsonToReadableText({
                        compilerOptions: compilerOptionsToConfigJson(compilerOptions),
                    }),
                });
            }
        }
    });

    verifyTsc({
        scenario: "noEmit",
        subScenario: "when project has strict true",
        commandLineArgs: ["-noEmit", "-p", `src/project`],
        fs: () =>
            loadProjectFromFiles({
                "/src/project/tsconfig.json": jsonToReadableText({
                    compilerOptions: {
                        incremental: true,
                        strict: true,
                    },
                }),
                "/src/project/class1.ts": `export class class1 {}`,
            }),
        edits: noChangeOnlyRuns,
        baselinePrograms: true,
    });

    verifyTsc({
        scenario: "noEmit",
        subScenario: "when isolatedDeclarations is true",
        commandLineArgs: ["-noEmit", "-p", `src/project`],
        fs: () =>
            loadProjectFromFiles({
                "/src/project/tsconfig.json": jsonToReadableText({
                    compilerOptions: {
                        declaration: true,
                        isolatedDeclarations: true,
                    },
                }),
                "/src/project/class1.ts": dedent`
                    function errorOnAssignmentBelowDecl(): void {}
                    errorOnAssignmentBelowDecl.a = "";
                `,
            }),
        edits: noChangeOnlyRuns,
        baselinePrograms: true,
    });
});
