import * as ts from "../../_namespaces/ts";

describe("unittests:: tsbuild:: commandLine::", () => {
    describe("different options::", () => {
        function withOptionChange(subScenario: string, ...options: readonly string[]): ts.TestTscEdit {
            return {
                subScenario,
                modifyFs: ts.noop,
                commandLineArgs: ["--b", "/src/project", "--verbose", ...options]
            };
        }
        function noChangeWithSubscenario(subScenario: string): ts.TestTscEdit {
            return { ...ts.noChangeRun, subScenario };
        }
        function withOptionChangeAndDiscrepancyExplanation(subScenario: string, option: string): ts.TestTscEdit {
            return {
                ...withOptionChange(subScenario, option),
                discrepancyExplanation: () => [
                    `Clean build tsbuildinfo will have compilerOptions with composite and ${option.replace(/\-/g, "")}`,
                    `Incremental build will detect that it doesnt need to rebuild so tsbuild info is from before which has option composite only`,
                ]
            };
        }
        function withEmitDeclarationOnlyChangeAndDiscrepancyExplanation(subScenario: string): ts.TestTscEdit {
            const edit = withOptionChangeAndDiscrepancyExplanation(subScenario, "--emitDeclarationOnly");
            const discrepancyExplanation = edit.discrepancyExplanation!;
            edit.discrepancyExplanation = () => [
                ...discrepancyExplanation(),
                `Clean build info does not have js section because its fresh build`,
                `Incremental build info has js section from old build`
            ];
            return edit;
        }
        function withOptionChangeAndExportExplanation(subScenario: string, ...options: readonly string[]): ts.TestTscEdit {
            return {
                ...withOptionChange(subScenario, ...options),
                discrepancyExplanation: ts.noChangeWithExportsDiscrepancyRun.discrepancyExplanation,
            };
        }
        function nochangeWithIncrementalDeclarationFromBeforeExplaination(): ts.TestTscEdit {
            return {
                ...ts.noChangeRun,
                discrepancyExplanation: () => [
                    `Clean build tsbuildinfo will have compilerOptions {}`,
                    `Incremental build will detect that it doesnt need to rebuild so tsbuild info is from before which has option declaration and declarationMap`,
                ],
            };
        }
        function nochangeWithIncrementalOutDeclarationFromBeforeExplaination(): ts.TestTscEdit {
            const edit = nochangeWithIncrementalDeclarationFromBeforeExplaination();
            const discrepancyExplanation = edit.discrepancyExplanation!;
            edit.discrepancyExplanation = () => [
                ...discrepancyExplanation(),
                `Clean build does not have dts bundle section`,
                `Incremental build contains the dts build section from before`,
            ];
            return edit;
        }
        function localChange(): ts.TestTscEdit {
            return {
                subScenario: "local change",
                modifyFs: fs => ts.replaceText(fs, "/src/project/a.ts", "Local = 1", "Local = 10"),
            };
        }
        function fs(options: ts.CompilerOptions) {
            return ts.loadProjectFromFiles({
                "/src/project/tsconfig.json": JSON.stringify({ compilerOptions: ts.compilerOptionsToConfigJson(options) }),
                "/src/project/a.ts": `export const a = 10;const aLocal = 10;`,
                "/src/project/b.ts": `export const b = 10;const bLocal = 10;`,
                "/src/project/c.ts": `import { a } from "./a";export const c = a;`,
                "/src/project/d.ts": `import { b } from "./b";export const d = b;`,
            });
        }
        ts.verifyTscWithEdits({
            scenario: "commandLine",
            subScenario: "different options",
            fs: () => fs({ composite: true }),
            commandLineArgs: ["--b", "/src/project", "--verbose"],
            edits: [
                withOptionChange("with sourceMap", "--sourceMap"),
                noChangeWithSubscenario("should re-emit only js so they dont contain sourcemap"),
                withOptionChangeAndDiscrepancyExplanation("with declaration should not emit anything", "--declaration"),
                ts.noChangeRun,
                withOptionChange("with declaration and declarationMap", "--declaration", "--declarationMap"),
                noChangeWithSubscenario("should re-emit only dts so they dont contain sourcemap"),
                withOptionChangeAndDiscrepancyExplanation("with emitDeclarationOnly should not emit anything", "--emitDeclarationOnly"),
                ts.noChangeRun,
                localChange(),
                withOptionChangeAndDiscrepancyExplanation("with declaration should not emit anything", "--declaration"),
                withOptionChange("with inlineSourceMap", "--inlineSourceMap"),
                withOptionChange("with sourceMap", "--sourceMap"),
            ],
            baselinePrograms: true,
        });
        ts.verifyTscWithEdits({
            scenario: "commandLine",
            subScenario: "different options with outFile",
            fs: () => fs({ composite: true, outFile: "../outFile.js", module: ts.ModuleKind.AMD }),
            commandLineArgs: ["--b", "/src/project", "--verbose"],
            edits: [
                withOptionChange("with sourceMap", "--sourceMap"),
                noChangeWithSubscenario("should re-emit only js so they dont contain sourcemap"),
                withOptionChangeAndDiscrepancyExplanation("with declaration should not emit anything", "--declaration"),
                ts.noChangeRun,
                withOptionChange("with declaration and declarationMap", "--declaration", "--declarationMap"),
                noChangeWithSubscenario("should re-emit only dts so they dont contain sourcemap"),
                withEmitDeclarationOnlyChangeAndDiscrepancyExplanation("with emitDeclarationOnly should not emit anything"),
                ts.noChangeRun,
                localChange(),
                withOptionChangeAndDiscrepancyExplanation("with declaration should not emit anything", "--declaration"),
                withOptionChange("with inlineSourceMap", "--inlineSourceMap"),
                withOptionChange("with sourceMap", "--sourceMap"),
            ],
            baselinePrograms: true,
        });
        ts.verifyTscWithEdits({
            scenario: "commandLine",
            subScenario: "different options with incremental",
            fs: () => fs({ incremental: true }),
            commandLineArgs: ["--b", "/src/project", "--verbose"],
            edits: [
                withOptionChangeAndExportExplanation("with sourceMap", "--sourceMap"),
                withOptionChangeAndExportExplanation("should re-emit only js so they dont contain sourcemap"),
                withOptionChange("with declaration, emit Dts and should not emit js", "--declaration"),
                withOptionChange("with declaration and declarationMap", "--declaration", "--declarationMap"),
                nochangeWithIncrementalDeclarationFromBeforeExplaination(),
                localChange(),
                withOptionChange("with declaration and declarationMap", "--declaration", "--declarationMap"),
                nochangeWithIncrementalDeclarationFromBeforeExplaination(),
                withOptionChange("with inlineSourceMap", "--inlineSourceMap"),
                withOptionChange("with sourceMap", "--sourceMap"),
                noChangeWithSubscenario("emit js files"),
                withOptionChange("with declaration and declarationMap", "--declaration", "--declarationMap"),
                withOptionChange("with declaration and declarationMap, should not re-emit", "--declaration", "--declarationMap"),
            ],
            baselinePrograms: true,
        });
        ts.verifyTscWithEdits({
            scenario: "commandLine",
            subScenario: "different options with incremental with outFile",
            fs: () => fs({ incremental: true, outFile: "../outFile.js", module: ts.ModuleKind.AMD }),
            commandLineArgs: ["--b", "/src/project", "--verbose"],
            edits: [
                withOptionChange("with sourceMap", "--sourceMap"),
                noChangeWithSubscenario("should re-emit only js so they dont contain sourcemap"),
                withOptionChange("with declaration, emit Dts and should not emit js", "--declaration"),
                withOptionChange("with declaration and declarationMap", "--declaration", "--declarationMap"),
                nochangeWithIncrementalOutDeclarationFromBeforeExplaination(),
                localChange(),
                withOptionChange("with declaration and declarationMap", "--declaration", "--declarationMap"),
                nochangeWithIncrementalOutDeclarationFromBeforeExplaination(),
                withOptionChange("with inlineSourceMap", "--inlineSourceMap"),
                withOptionChange("with sourceMap", "--sourceMap"),
                noChangeWithSubscenario("emit js files"),
                withOptionChange("with declaration and declarationMap", "--declaration", "--declarationMap"),
                withOptionChange("with declaration and declarationMap, should not re-emit", "--declaration", "--declarationMap"),
            ],
            baselinePrograms: true,
        });
    });
    describe("emitDeclarationOnly::", () => {
        function fs(options: ts.CompilerOptions) {
            return ts.loadProjectFromFiles({
                "/src/project1/src/tsconfig.json": JSON.stringify({
                    compilerOptions: ts.compilerOptionsToConfigJson(options),
                }),
                "/src/project1/src/a.ts": `export const a = 10;const aLocal = 10;`,
                "/src/project1/src/b.ts": `export const b = 10;const bLocal = 10;`,
                "/src/project1/src/c.ts": `import { a } from "./a";export const c = a;`,
                "/src/project1/src/d.ts": `import { b } from "./b";export const d = b;`,
                "/src/project2/src/tsconfig.json": JSON.stringify({
                    compilerOptions: ts.compilerOptionsToConfigJson(options),
                    references: [{ path: "../../project1/src" }],
                }),
                "/src/project2/src/e.ts": `export const e = 10;`,
                "/src/project2/src/f.ts": `import { a } from "${options.outFile ? "a" : "../../project1/src/a"}"; export const f = a;`,
                "/src/project2/src/g.ts": `import { b } from "${options.outFile ? "b" : "../../project1/src/b"}"; export const g = b;`,
            });
        }
        function verifyWithIncremental(options: ts.CompilerOptions) {
            ts.verifyTscWithEdits({
                scenario: "commandLine",
                subScenario: subScenario("emitDeclarationOnly on commandline"),
                fs: () => fs(options),
                commandLineArgs: ["--b", "/src/project2/src", "--verbose", "--emitDeclarationOnly"],
                edits: [
                    ts.noChangeRun,
                    {
                        subScenario: "local change",
                        modifyFs: fs => ts.appendText(fs, "/src/project1/src/a.ts", "const aa = 10;"),
                    },
                    {
                        subScenario: "non local change",
                        modifyFs: fs => ts.appendText(fs, "/src/project1/src/a.ts", "export const aaa = 10;"),
                    },
                    {
                        subScenario: "emit js files",
                        modifyFs: ts.noop,
                        commandLineArgs: ["--b", "/src/project2/src", "--verbose"],
                    },
                    {
                        ...ts.noChangeRun,
                        discrepancyExplanation: () => [
                            `Clean build tsbuildinfo for both projects will have compilerOptions with composite and emitDeclarationOnly`,
                            `Incremental build will detect that it doesnt need to rebuild so tsbuild info for projects is from before which has option composite only`,
                        ]
                    },
                    {
                        subScenario: "js emit with change without emitDeclarationOnly",
                        modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "const alocal = 10;"),
                        commandLineArgs: ["--b", "/src/project2/src", "--verbose"],
                    },
                    {
                        subScenario: "local change",
                        modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "const aaaa = 10;"),
                        // --out without composite doesnt emit buildInfo without emitting program so it wouldnt have project2 tsbuildInfo so no mismatch
                        discrepancyExplanation: options.incremental && options.outFile ? undefined : () => [
                            `Clean build tsbuildinfo for project2 will have compilerOptions with composite and emitDeclarationOnly`,
                            `Incremental build will detect that it doesnt need to rebuild project2 so tsbuildinfo for it is from before which has option composite only`,
                        ],
                    },
                    {
                        subScenario: "non local change",
                        modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "export const aaaaa = 10;"),
                    },
                    {
                        subScenario: "js emit with change without emitDeclarationOnly",
                        modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "export const a2 = 10;"),
                        commandLineArgs: ["--b", "/src/project2/src", "--verbose"],
                    },
                ],
                baselinePrograms: true,
            });
            ts.verifyTscWithEdits({
                scenario: "commandLine",
                subScenario: subScenario("emitDeclarationOnly false on commandline"),
                fs: () => fs({ ...options, emitDeclarationOnly: true }),
                commandLineArgs: ["--b", "/src/project2/src", "--verbose"],
                edits: [
                    ts.noChangeRun,
                    {
                        subScenario: "change",
                        modifyFs: fs => ts.appendText(fs, "/src/project1/src/a.ts", "const aa = 10;"),
                    },
                    {
                        subScenario: "emit js files",
                        modifyFs: ts.noop,
                        commandLineArgs: ["--b", "/src/project2/src", "--verbose", "--emitDeclarationOnly", "false"],
                    },
                    {
                        ...ts.noChangeRun,
                        discrepancyExplanation: () => [
                            `Clean build tsbuildinfo for both projects will have compilerOptions with composite and emitDeclarationOnly`,
                            `Incremental build will detect that it doesnt need to rebuild so tsbuild info for projects is from before which has option composite as true but emitDeclrationOnly as false`,
                        ]
                    },
                    {
                        subScenario: "no change run with js emit",
                        modifyFs: ts.noop,
                        commandLineArgs: ["--b", "/src/project2/src", "--verbose", "--emitDeclarationOnly", "false"],
                    },
                    {
                        subScenario: "js emit with change",
                        modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "const blocal = 10;"),
                        commandLineArgs: ["--b", "/src/project2/src", "--verbose", "--emitDeclarationOnly", "false"],
                    },
                ],
                baselinePrograms: true,
            });
            function subScenario(text: string) {
                return `${text}${options.composite ? "" : " with declaration and incremental"}${options.outFile ? " with outFile" : ""}`;
            }
        }
        verifyWithIncremental({ composite: true });
        verifyWithIncremental({ incremental: true, declaration: true });
        verifyWithIncremental({ composite: true, outFile: "../outFile.js", module: ts.ModuleKind.AMD });
        verifyWithIncremental({ incremental: true, declaration: true, outFile: "../outFile.js", module: ts.ModuleKind.AMD });

        ts.verifyTscWithEdits({
            scenario: "commandLine",
            subScenario: "emitDeclarationOnly on commandline with declaration",
            fs: () => fs({ declaration: true }),
            commandLineArgs: ["--b", "/src/project2/src", "--verbose", "--emitDeclarationOnly"],
            edits: [
                ts.noChangeRun,
                {
                    subScenario: "local change",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/a.ts", "const aa = 10;"),
                },
                {
                    subScenario: "non local change",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/a.ts", "export const aaa = 10;"),
                },
                {
                    subScenario: "emit js files",
                    modifyFs: ts.noop,
                    commandLineArgs: ["--b", "/src/project2/src", "--verbose"],
                },
                ts.noChangeRun,
                {
                    subScenario: "js emit with change without emitDeclarationOnly",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "const alocal = 10;"),
                    commandLineArgs: ["--b", "/src/project2/src", "--verbose"],
                },
                {
                    subScenario: "local change",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "const aaaa = 10;"),
                },
                {
                    subScenario: "non local change",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "export const aaaaa = 10;"),
                },
                {
                    subScenario: "js emit with change without emitDeclarationOnly",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "export const a2 = 10;"),
                    commandLineArgs: ["--b", "/src/project2/src", "--verbose"],
                },
            ],
            baselinePrograms: true,
        });

        ts.verifyTscWithEdits({
            scenario: "commandLine",
            subScenario: "emitDeclarationOnly false on commandline with declaration",
            fs: () => fs({ declaration: true, emitDeclarationOnly: true }),
            commandLineArgs: ["--b", "/src/project2/src", "--verbose"],
            edits: [
                ts.noChangeRun,
                {
                    subScenario: "change",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/a.ts", "const aa = 10;"),
                },
                {
                    subScenario: "emit js files",
                    modifyFs: ts.noop,
                    commandLineArgs: ["--b", "/src/project2/src", "--verbose", "--emitDeclarationOnly", "false"],
                },
                ts.noChangeRun,
                {
                    subScenario: "no change run with js emit",
                    modifyFs: ts.noop,
                    commandLineArgs: ["--b", "/src/project2/src", "--verbose", "--emitDeclarationOnly", "false"],
                },
                {
                    subScenario: "js emit with change",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "const blocal = 10;"),
                    commandLineArgs: ["--b", "/src/project2/src", "--verbose", "--emitDeclarationOnly", "false"],
                },
            ],
            baselinePrograms: true,
        });

        ts.verifyTscWithEdits({
            scenario: "commandLine",
            subScenario: "emitDeclarationOnly on commandline with declaration with outFile",
            fs: () => fs({ declaration: true, outFile: "../outFile.js", module: ts.ModuleKind.AMD }),
            commandLineArgs: ["--b", "/src/project2/src", "--verbose", "--emitDeclarationOnly"],
            edits: [
                ts.noChangeRun,
                {
                    subScenario: "local change",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/a.ts", "const aa = 10;"),
                },
                {
                    subScenario: "non local change",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/a.ts", "export const aaa = 10;"),
                },
                {
                    subScenario: "emit js files",
                    modifyFs: ts.noop,
                    commandLineArgs: ["--b", "/src/project2/src", "--verbose"],
                },
                ts.noChangeRun,
                {
                    subScenario: "js emit with change without emitDeclarationOnly",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "const alocal = 10;"),
                    commandLineArgs: ["--b", "/src/project2/src", "--verbose"],
                },
                {
                    subScenario: "local change",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "const aaaa = 10;"),
                },
                {
                    subScenario: "non local change",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "export const aaaaa = 10;"),
                },
                {
                    subScenario: "js emit with change without emitDeclarationOnly",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "export const a2 = 10;"),
                    commandLineArgs: ["--b", "/src/project2/src", "--verbose"],
                },
            ],
            baselinePrograms: true,
        });

        ts.verifyTscWithEdits({
            scenario: "commandLine",
            subScenario: "emitDeclarationOnly false on commandline with declaration with outFile",
            fs: () => fs({ declaration: true, emitDeclarationOnly: true, outFile: "../outFile.js", module: ts.ModuleKind.AMD }),
            commandLineArgs: ["--b", "/src/project2/src", "--verbose"],
            edits: [
                ts.noChangeRun,
                {
                    subScenario: "change",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/a.ts", "const aa = 10;"),
                },
                {
                    subScenario: "emit js files",
                    modifyFs: ts.noop,
                    commandLineArgs: ["--b", "/src/project2/src", "--verbose", "--emitDeclarationOnly", "false"],
                },
                ts.noChangeRun,
                {
                    subScenario: "no change run with js emit",
                    modifyFs: ts.noop,
                    commandLineArgs: ["--b", "/src/project2/src", "--verbose", "--emitDeclarationOnly", "false"],
                },
                {
                    subScenario: "js emit with change",
                    modifyFs: fs => ts.appendText(fs, "/src/project1/src/b.ts", "const blocal = 10;"),
                    commandLineArgs: ["--b", "/src/project2/src", "--verbose", "--emitDeclarationOnly", "false"],
                },
            ],
            baselinePrograms: true,
        });
    });
});