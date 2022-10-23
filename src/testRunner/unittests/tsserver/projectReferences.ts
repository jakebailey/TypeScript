import * as ts from "../../_namespaces/ts";
import { createHostWithSolutionBuild } from "./helpers";

describe("unittests:: tsserver:: with project references and tsbuild", () => {
    describe("with container project", () => {
        function getProjectFiles(project: string): [ts.projectSystem.File, ts.projectSystem.File] {
            return [
                ts.TestFSWithWatch.getTsBuildProjectFile(project, "tsconfig.json"),
                ts.TestFSWithWatch.getTsBuildProjectFile(project, "index.ts"),
            ];
        }

        const project = "container";
        const containerLib = getProjectFiles("container/lib");
        const containerExec = getProjectFiles("container/exec");
        const containerCompositeExec = getProjectFiles("container/compositeExec");
        const containerConfig = ts.TestFSWithWatch.getTsBuildProjectFile(project, "tsconfig.json");
        const files = [ts.projectSystem.libFile, ...containerLib, ...containerExec, ...containerCompositeExec, containerConfig];

        it("does not error on container only project", () => {
            const host = createHostWithSolutionBuild(files, [containerConfig.path]);

            // Open external project for the folder
            const session = ts.projectSystem.createSession(host, { logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
            const service = session.getProjectService();
            service.openExternalProjects([{
                projectFileName: ts.TestFSWithWatch.getTsBuildProjectFilePath(project, project),
                rootFiles: files.map(f => ({ fileName: f.path })),
                options: {}
            }]);
            files.forEach(f => {
                const args: ts.projectSystem.protocol.FileRequestArgs = {
                    file: f.path,
                    projectFileName: ts.endsWith(f.path, "tsconfig.json") ? f.path : undefined
                };
                session.executeCommandSeq<ts.projectSystem.protocol.SyntacticDiagnosticsSyncRequest>({
                    command: ts.projectSystem.protocol.CommandTypes.SyntacticDiagnosticsSync,
                    arguments: args
                });
                session.executeCommandSeq<ts.projectSystem.protocol.SemanticDiagnosticsSyncRequest>({
                    command: ts.projectSystem.protocol.CommandTypes.SemanticDiagnosticsSync,
                    arguments: args
                });
            });
            const containerProject = service.configuredProjects.get(containerConfig.path)!;
            session.executeCommandSeq<ts.projectSystem.protocol.CompilerOptionsDiagnosticsRequest>({
                command: ts.projectSystem.protocol.CommandTypes.CompilerOptionsDiagnosticsFull,
                arguments: { projectFileName: containerProject.projectName }
            });
            ts.projectSystem.baselineTsserverLogs("projectReferences", `does not error on container only project`, session);
        });

        it("can successfully find references with --out options", () => {
            const host = createHostWithSolutionBuild(files, [containerConfig.path]);
            const session = ts.projectSystem.createSession(host, { logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
            ts.projectSystem.openFilesForSession([containerCompositeExec[1]], session);
            const myConstStart = ts.projectSystem.protocolLocationFromSubstring(containerCompositeExec[1].content, "myConst");
            session.executeCommandSeq<ts.projectSystem.protocol.RenameRequest>({
                command: ts.projectSystem.protocol.CommandTypes.Rename,
                arguments: { file: containerCompositeExec[1].path, ...myConstStart }
            });

            ts.projectSystem.baselineTsserverLogs("projectReferences", `can successfully find references with out option`, session);
        });

        it("ancestor and project ref management", () => {
            const tempFile: ts.projectSystem.File = {
                path: `/user/username/projects/temp/temp.ts`,
                content: "let x = 10"
            };
            const host = createHostWithSolutionBuild(files.concat([tempFile]), [containerConfig.path]);
            const session = ts.projectSystem.createSession(host, { logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
            ts.projectSystem.openFilesForSession([containerCompositeExec[1]], session);
            const service = session.getProjectService();

            // Open temp file and verify all projects alive
            ts.projectSystem.openFilesForSession([tempFile], session);

            // Ref projects are loaded after as part of this command
            const locationOfMyConst = ts.projectSystem.protocolLocationFromSubstring(containerCompositeExec[1].content, "myConst");
            session.executeCommandSeq<ts.projectSystem.protocol.RenameRequest>({
                command: ts.projectSystem.protocol.CommandTypes.Rename,
                arguments: {
                    file: containerCompositeExec[1].path,
                    ...locationOfMyConst
                }
            });

            // Open temp file and verify all projects alive
            service.closeClientFile(tempFile.path);
            ts.projectSystem.openFilesForSession([tempFile], session);

            // Close all files and open temp file, only inferred project should be alive
            service.closeClientFile(containerCompositeExec[1].path);
            service.closeClientFile(tempFile.path);
            ts.projectSystem.openFilesForSession([tempFile], session);
            ts.projectSystem.baselineTsserverLogs("projectReferences", `ancestor and project ref management`, session);
        });
    });

    describe("when root file is file from referenced project", () => {
        function verify(disableSourceOfProjectReferenceRedirect: boolean) {
            const projectLocation = `/user/username/projects/project`;
            const commonConfig: ts.projectSystem.File = {
                path: `${projectLocation}/src/common/tsconfig.json`,
                content: JSON.stringify({
                    compilerOptions: {
                        composite: true,
                        declarationMap: true,
                        outDir: "../../out",
                        baseUrl: "..",
                        disableSourceOfProjectReferenceRedirect
                    },
                    include: ["./**/*"]
                })
            };
            const keyboardTs: ts.projectSystem.File = {
                path: `${projectLocation}/src/common/input/keyboard.ts`,
                content: `function bar() { return "just a random function so .d.ts location doesnt match"; }
export function evaluateKeyboardEvent() { }`
            };
            const keyboardTestTs: ts.projectSystem.File = {
                path: `${projectLocation}/src/common/input/keyboard.test.ts`,
                content: `import { evaluateKeyboardEvent } from 'common/input/keyboard';
function testEvaluateKeyboardEvent() {
    return evaluateKeyboardEvent();
}
`
            };
            const srcConfig: ts.projectSystem.File = {
                path: `${projectLocation}/src/tsconfig.json`,
                content: JSON.stringify({
                    compilerOptions: {
                        composite: true,
                        declarationMap: true,
                        outDir: "../out",
                        baseUrl: ".",
                        paths: {
                            "common/*": ["./common/*"],
                        },
                        tsBuildInfoFile: "../out/src.tsconfig.tsbuildinfo",
                        disableSourceOfProjectReferenceRedirect
                    },
                    include: ["./**/*"],
                    references: [
                        { path: "./common" }
                    ]
                })
            };
            const terminalTs: ts.projectSystem.File = {
                path: `${projectLocation}/src/terminal.ts`,
                content: `import { evaluateKeyboardEvent } from 'common/input/keyboard';
function foo() {
    return evaluateKeyboardEvent();
}
`
            };
            const host = createHostWithSolutionBuild(
                [commonConfig, keyboardTs, keyboardTestTs, srcConfig, terminalTs, ts.projectSystem.libFile],
                [srcConfig.path]
            );
            const session = ts.projectSystem.createSession(host, { logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
            ts.projectSystem.openFilesForSession([keyboardTs, terminalTs], session);

            const searchStr = "evaluateKeyboardEvent";
            const importStr = `import { evaluateKeyboardEvent } from 'common/input/keyboard';`;
            const result = session.executeCommandSeq<ts.projectSystem.protocol.ReferencesRequest>({
                command: ts.projectSystem.protocol.CommandTypes.References,
                arguments: ts.projectSystem.protocolFileLocationFromSubstring(keyboardTs, searchStr)
            }).response as ts.projectSystem.protocol.ReferencesResponseBody;
            assert.deepEqual(result, {
                refs: [
                    ts.projectSystem.makeReferenceItem({
                        file: keyboardTs,
                        text: searchStr,
                        contextText: `export function evaluateKeyboardEvent() { }`,
                        isDefinition: true,
                        lineText: `export function evaluateKeyboardEvent() { }`
                    }),
                    ts.projectSystem.makeReferenceItem({
                        file: keyboardTestTs,
                        text: searchStr,
                        contextText: importStr,
                        isDefinition: false,
                        isWriteAccess: true,
                        lineText: importStr
                    }),
                    ts.projectSystem.makeReferenceItem({
                        file: keyboardTestTs,
                        text: searchStr,
                        options: { index: 1 },
                        isDefinition: false,
                        lineText: `    return evaluateKeyboardEvent();`
                    }),
                    ts.projectSystem.makeReferenceItem({
                        file: terminalTs,
                        text: searchStr,
                        contextText: importStr,
                        isDefinition: false,
                        isWriteAccess: true,
                        lineText: importStr
                    }),
                    ts.projectSystem.makeReferenceItem({
                        file: terminalTs,
                        text: searchStr,
                        options: { index: 1 },
                        isDefinition: false,
                        lineText: `    return evaluateKeyboardEvent();`
                    }),
                ],
                symbolName: searchStr,
                symbolStartOffset: ts.projectSystem.protocolLocationFromSubstring(keyboardTs.content, searchStr).offset,
                symbolDisplayString: "function evaluateKeyboardEvent(): void"
            });
            ts.projectSystem.baselineTsserverLogs("projectReferences", `root file is file from referenced project${disableSourceOfProjectReferenceRedirect ? " and using declaration maps" : ""}`, session);
        }

        it(`when using declaration file maps to navigate between projects`, () => {
            verify(/*disableSourceOfProjectReferenceRedirect*/ true);
        });
        it(`when using original source files in the project`, () => {
            verify(/*disableSourceOfProjectReferenceRedirect*/ false);
        });
    });

    it("reusing d.ts files from composite and non composite projects", () => {
        const configA: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/compositea/tsconfig.json`,
            content: JSON.stringify({
                compilerOptions: {
                    composite: true,
                    outDir: "../dist/",
                    rootDir: "../",
                    baseUrl: "../",
                    paths: { "@ref/*": ["./dist/*"] }
                }
            })
        };
        const aTs: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/compositea/a.ts`,
            content: `import { b } from "@ref/compositeb/b";`
        };
        const a2Ts: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/compositea/a2.ts`,
            content: `export const x = 10;`
        };
        const configB: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/compositeb/tsconfig.json`,
            content: configA.content
        };
        const bTs: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/compositeb/b.ts`,
            content: "export function b() {}"
        };
        const bDts: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/dist/compositeb/b.d.ts`,
            content: "export declare function b(): void;"
        };
        const configC: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/compositec/tsconfig.json`,
            content: JSON.stringify({
                compilerOptions: {
                    composite: true,
                    outDir: "../dist/",
                    rootDir: "../",
                    baseUrl: "../",
                    paths: { "@ref/*": ["./*"] }
                },
                references: [{ path: "../compositeb" }]
            })
        };
        const cTs: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/compositec/c.ts`,
            content: aTs.content
        };
        const files = [ts.projectSystem.libFile, aTs, a2Ts, configA, bDts, bTs, configB, cTs, configC];
        const host = ts.projectSystem.createServerHost(files);
        const service = ts.projectSystem.createProjectService(host);
        service.openClientFile(aTs.path);
        service.checkNumberOfProjects({ configuredProjects: 1 });

        // project A referencing b.d.ts without project reference
        const projectA = service.configuredProjects.get(configA.path)!;
        assert.isDefined(projectA);
        ts.projectSystem.checkProjectActualFiles(projectA, [aTs.path, a2Ts.path, bDts.path, ts.projectSystem.libFile.path, configA.path]);

        // reuses b.d.ts but sets the path and resolved path since projectC has project references
        // as the real resolution was to b.ts
        service.openClientFile(cTs.path);
        service.checkNumberOfProjects({ configuredProjects: 2 });
        const projectC = service.configuredProjects.get(configC.path)!;
        ts.projectSystem.checkProjectActualFiles(projectC, [cTs.path, bTs.path, ts.projectSystem.libFile.path, configC.path]);

        // Now new project for project A tries to reuse b but there is no filesByName mapping for b's source location
        host.writeFile(a2Ts.path, `${a2Ts.content}export const y = 30;`);
        assert.isTrue(projectA.dirty);
        projectA.updateGraph();
    });

    describe("when references are monorepo like with symlinks", () => {
        interface Packages {
            bPackageJson: ts.projectSystem.File;
            aTest: ts.projectSystem.File;
            bFoo: ts.projectSystem.File;
            bBar: ts.projectSystem.File;
            bSymlink: ts.projectSystem.SymLink;
        }
        function verifySymlinkScenario(scenario: string, packages: () => Packages) {
            describe(`${scenario}: when solution is not built`, () => {
                it("with preserveSymlinks turned off", () => {
                    verifySession(scenario, packages(), /*alreadyBuilt*/ false, {});
                });

                it("with preserveSymlinks turned on", () => {
                    verifySession(scenario, packages(), /*alreadyBuilt*/ false, { preserveSymlinks: true });
                });
            });

            describe(`${scenario}: when solution is already built`, () => {
                it("with preserveSymlinks turned off", () => {
                    verifySession(scenario, packages(), /*alreadyBuilt*/ true, {});
                });

                it("with preserveSymlinks turned on", () => {
                    verifySession(scenario, packages(), /*alreadyBuilt*/ true, { preserveSymlinks: true });
                });
            });
        }

        function verifySession(scenario: string, { bPackageJson, aTest, bFoo, bBar, bSymlink }: Packages, alreadyBuilt: boolean, extraOptions: ts.CompilerOptions) {
            const aConfig = config("A", extraOptions, ["../B"]);
            const bConfig = config("B", extraOptions);
            const files = [ts.projectSystem.libFile, bPackageJson, aConfig, bConfig, aTest, bFoo, bBar, bSymlink];
            const host = alreadyBuilt ?
                createHostWithSolutionBuild(files, [aConfig.path]) :
                ts.projectSystem.createServerHost(files);

            // Create symlink in node module
            const session = ts.projectSystem.createSession(host, { canUseEvents: true, logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
            ts.projectSystem.openFilesForSession([aTest], session);
            ts.projectSystem.verifyGetErrRequest({ session, host, files: [aTest] });
            session.executeCommandSeq<ts.projectSystem.protocol.UpdateOpenRequest>({
                command: ts.projectSystem.protocol.CommandTypes.UpdateOpen,
                arguments: {
                    changedFiles: [{
                        fileName: aTest.path,
                        textChanges: [{
                            newText: "\n",
                            start: { line: 5, offset: 1 },
                            end: { line: 5, offset: 1 }
                        }]
                    }]
                }
            });
            ts.projectSystem.verifyGetErrRequest({ session, host, files: [aTest] });
            ts.projectSystem.baselineTsserverLogs("projectReferences", `monorepo like with symlinks ${scenario} and solution is ${alreadyBuilt ? "built" : "not built"}${extraOptions.preserveSymlinks ? " with preserveSymlinks" : ""}`, session);
        }

        function config(packageName: string, extraOptions: ts.CompilerOptions, references?: string[]): ts.projectSystem.File {
            return {
                path: `${ts.tscWatch.projectRoot}/packages/${packageName}/tsconfig.json`,
                content: JSON.stringify({
                    compilerOptions: {
                        outDir: "lib",
                        rootDir: "src",
                        composite: true,
                        ...extraOptions
                    },
                    include: ["src"],
                    ...(references ? { references: references.map(path => ({ path })) } : {})
                })
            };
        }

        function file(packageName: string, fileName: string, content: string): ts.projectSystem.File {
            return {
                path: `${ts.tscWatch.projectRoot}/packages/${packageName}/src/${fileName}`,
                content
            };
        }

        function verifyMonoRepoLike(scope = "") {
            verifySymlinkScenario(`when packageJson has types field and has index.ts${scope ? " with scoped package" : ""}`, () => ({
                bPackageJson: {
                    path: `${ts.tscWatch.projectRoot}/packages/B/package.json`,
                    content: JSON.stringify({
                        main: "lib/index.js",
                        types: "lib/index.d.ts"
                    })
                },
                aTest: file("A", "index.ts", `import { foo } from '${scope}b';
import { bar } from '${scope}b/lib/bar';
foo();
bar();
`),
                bFoo: file("B", "index.ts", `export function foo() { }`),
                bBar: file("B", "bar.ts", `export function bar() { }`),
                bSymlink: {
                    path: `${ts.tscWatch.projectRoot}/node_modules/${scope}b`,
                    symLink: `${ts.tscWatch.projectRoot}/packages/B`
                }
            }));

            verifySymlinkScenario(`when referencing file from subFolder${scope ? " with scoped package" : ""}`, () => ({
                bPackageJson: {
                    path: `${ts.tscWatch.projectRoot}/packages/B/package.json`,
                    content: "{}"
                },
                aTest: file("A", "test.ts", `import { foo } from '${scope}b/lib/foo';
import { bar } from '${scope}b/lib/bar/foo';
foo();
bar();
`),
                bFoo: file("B", "foo.ts", `export function foo() { }`),
                bBar: file("B", "bar/foo.ts", `export function bar() { }`),
                bSymlink: {
                    path: `${ts.tscWatch.projectRoot}/node_modules/${scope}b`,
                    symLink: `${ts.tscWatch.projectRoot}/packages/B`
                }
            }));
        }

        describe("when package is not scoped", () => {
            verifyMonoRepoLike();
        });
        describe("when package is scoped", () => {
            verifyMonoRepoLike("@issue/");
        });
    });

    it("when the referenced projects have allowJs and emitDeclarationOnly", () => {
        const compositeConfig: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/packages/emit-composite/tsconfig.json`,
            content: JSON.stringify({
                compilerOptions: {
                    composite: true,
                    allowJs: true,
                    emitDeclarationOnly: true,
                    outDir: "lib",
                    rootDir: "src"
                },
                include: ["src"]
            })
        };
        const compositePackageJson: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/packages/emit-composite/package.json`,
            content: JSON.stringify({
                name: "emit-composite",
                version: "1.0.0",
                main: "src/index.js",
                typings: "lib/index.d.ts"
            })
        };
        const compositeIndex: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/packages/emit-composite/src/index.js`,
            content: `const testModule = require('./testModule');
module.exports = {
    ...testModule
}`
        };
        const compositeTestModule: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/packages/emit-composite/src/testModule.js`,
            content: `/**
 * @param {string} arg
 */
 const testCompositeFunction = (arg) => {
}
module.exports = {
    testCompositeFunction
}`
        };
        const consumerConfig: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/packages/consumer/tsconfig.json`,
            content: JSON.stringify({
                include: ["src"],
                references: [{ path: "../emit-composite" }]
            })
        };
        const consumerIndex: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/packages/consumer/src/index.ts`,
            content: `import { testCompositeFunction } from 'emit-composite';
testCompositeFunction('why hello there');
testCompositeFunction('why hello there', 42);`
        };
        const symlink: ts.projectSystem.SymLink = {
            path: `${ts.tscWatch.projectRoot}/node_modules/emit-composite`,
            symLink: `${ts.tscWatch.projectRoot}/packages/emit-composite`
        };
        const host = ts.projectSystem.createServerHost([ts.projectSystem.libFile, compositeConfig, compositePackageJson, compositeIndex, compositeTestModule, consumerConfig, consumerIndex, symlink], { useCaseSensitiveFileNames: true });
        const session = ts.projectSystem.createSession(host, { canUseEvents: true, logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
        ts.projectSystem.openFilesForSession([consumerIndex], session);
        ts.projectSystem.verifyGetErrRequest({ host, session, files: [consumerIndex] });
        ts.projectSystem.baselineTsserverLogs("projectReferences", `when the referenced projects have allowJs and emitDeclarationOnly`, session);
    });

    it("when finding local reference doesnt load ancestor/sibling projects", () => {
        const solutionLocation = "/user/username/projects/solution";
        const solution: ts.projectSystem.File = {
            path: `${solutionLocation}/tsconfig.json`,
            content: JSON.stringify({
                files: [],
                include: [],
                references: [
                    { path: "./compiler" },
                    { path: "./services" },
                ]
            })
        };
        const compilerConfig: ts.projectSystem.File = {
            path: `${solutionLocation}/compiler/tsconfig.json`,
            content: JSON.stringify({
                compilerOptions: {
                    composite: true,
                    module: "none"
                },
                files: ["./types.ts", "./program.ts"]
            })
        };
        const typesFile: ts.projectSystem.File = {
            path: `${solutionLocation}/compiler/types.ts`,
            content: `
                namespace ts {
                    export interface Program {
                        getSourceFiles(): string[];
                    }
                }`
        };
        const programFile: ts.projectSystem.File = {
            path: `${solutionLocation}/compiler/program.ts`,
            content: `
                namespace ts {
                    export const program: Program = {
                        getSourceFiles: () => [getSourceFile()]
                    };
                    function getSourceFile() { return "something"; }
                }`
        };
        const servicesConfig: ts.projectSystem.File = {
            path: `${solutionLocation}/services/tsconfig.json`,
            content: JSON.stringify({
                compilerOptions: {
                    composite: true
                },
                files: ["./services.ts"],
                references: [
                    { path: "../compiler" }
                ]
            })
        };
        const servicesFile: ts.projectSystem.File = {
            path: `${solutionLocation}/services/services.ts`,
            content: `
                namespace ts {
                    const result = program.getSourceFiles();
                }`
        };

        const files = [ts.projectSystem.libFile, solution, compilerConfig, typesFile, programFile, servicesConfig, servicesFile, ts.projectSystem.libFile];
        const host = ts.projectSystem.createServerHost(files);
        const session = ts.projectSystem.createSession(host, { logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
        ts.projectSystem.openFilesForSession([programFile], session);

        // Find all references for getSourceFile
        // Shouldnt load more projects
        session.executeCommandSeq<ts.projectSystem.protocol.ReferencesRequest>({
            command: ts.projectSystem.protocol.CommandTypes.References,
            arguments: ts.projectSystem.protocolFileLocationFromSubstring(programFile, "getSourceFile", { index: 1 })
        });

        // Find all references for getSourceFiles
        // Should load more projects
        session.executeCommandSeq<ts.projectSystem.protocol.ReferencesRequest>({
            command: ts.projectSystem.protocol.CommandTypes.References,
            arguments: ts.projectSystem.protocolFileLocationFromSubstring(programFile, "getSourceFiles")
        });
        ts.projectSystem.baselineTsserverLogs("projectReferences", `finding local reference doesnt load ancestor/sibling projects`, session);
    });

    it("when finding references in overlapping projects", () => {
        const solutionLocation = "/user/username/projects/solution";
        const solutionConfig: ts.projectSystem.File = {
            path: `${solutionLocation}/tsconfig.json`,
            content: JSON.stringify({
                files: [],
                include: [],
                references: [
                    { path: "./a" },
                    { path: "./b" },
                    { path: "./c" },
                    { path: "./d" },
                ]
            })
        };
        const aConfig: ts.projectSystem.File = {
            path: `${solutionLocation}/a/tsconfig.json`,
            content: JSON.stringify({
                compilerOptions: {
                    composite: true,
                    module: "none"
                },
                files: ["./index.ts"]
            })
        };
        const aFile: ts.projectSystem.File = {
            path: `${solutionLocation}/a/index.ts`,
            content: `
                export interface I {
                    M(): void;
                }`
        };

        const bConfig: ts.projectSystem.File = {
            path: `${solutionLocation}/b/tsconfig.json`,
            content: JSON.stringify({
                compilerOptions: {
                    composite: true
                },
                files: ["./index.ts"],
                references: [
                    { path: "../a" }
                ]
            })
        };
        const bFile: ts.projectSystem.File = {
            path: `${solutionLocation}/b/index.ts`,
            content: `
                import { I } from "../a";

                export class B implements I {
                    M() {}
                }`
        };

        const cConfig: ts.projectSystem.File = {
            path: `${solutionLocation}/c/tsconfig.json`,
            content: JSON.stringify({
                compilerOptions: {
                    composite: true
                },
                files: ["./index.ts"],
                references: [
                    { path: "../b" }
                ]
            })
        };
        const cFile: ts.projectSystem.File = {
            path: `${solutionLocation}/c/index.ts`,
            content: `
                import { I } from "../a";
                import { B } from "../b";

                export const C: I = new B();
                `
        };

        const dConfig: ts.projectSystem.File = {
            path: `${solutionLocation}/d/tsconfig.json`,
            content: JSON.stringify({
                compilerOptions: {
                    composite: true
                },
                files: ["./index.ts"],
                references: [
                    { path: "../c" }
                ]
            })
        };
        const dFile: ts.projectSystem.File = {
            path: `${solutionLocation}/d/index.ts`,
            content: `
                import { I } from "../a";
                import { C } from "../c";

                export const D: I = C;
                `
        };

        const files = [ts.projectSystem.libFile, solutionConfig, aConfig, aFile, bConfig, bFile, cConfig, cFile, dConfig, dFile, ts.projectSystem.libFile];
        const host = ts.projectSystem.createServerHost(files);
        const session = ts.projectSystem.createSession(host, { logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
        ts.projectSystem.openFilesForSession([bFile], session);

        // The first search will trigger project loads
        session.executeCommandSeq<ts.projectSystem.protocol.ReferencesRequest>({
            command: ts.projectSystem.protocol.CommandTypes.References,
            arguments: ts.projectSystem.protocolFileLocationFromSubstring(bFile, "I", { index: 1 })
        });

        // The second search starts with the projects already loaded
        // Formerly, this would search some projects multiple times
        session.executeCommandSeq<ts.projectSystem.protocol.ReferencesRequest>({
            command: ts.projectSystem.protocol.CommandTypes.References,
            arguments: ts.projectSystem.protocolFileLocationFromSubstring(bFile, "I", { index: 1 })
        });

        ts.projectSystem.baselineTsserverLogs("projectReferences", `finding references in overlapping projects`, session);
    });

    describe("special handling of localness of the definitions for findAllRefs", () => {
        function verify(scenario: string, definition: string, usage: string, referenceTerm: string) {
            it(scenario, () => {
                const solutionLocation = "/user/username/projects/solution";
                const solution: ts.projectSystem.File = {
                    path: `${solutionLocation}/tsconfig.json`,
                    content: JSON.stringify({
                        files: [],
                        references: [
                            { path: "./api" },
                            { path: "./app" },
                        ]
                    })
                };
                const apiConfig: ts.projectSystem.File = {
                    path: `${solutionLocation}/api/tsconfig.json`,
                    content: JSON.stringify({
                        compilerOptions: {
                            composite: true,
                            outDir: "dist",
                            rootDir: "src",
                        },
                        include: ["src"],
                        references: [{ path: "../shared" }]
                    })
                };
                const apiFile: ts.projectSystem.File = {
                    path: `${solutionLocation}/api/src/server.ts`,
                    content: `import * as shared from "../../shared/dist";
${usage}`
                };
                const appConfig: ts.projectSystem.File = {
                    path: `${solutionLocation}/app/tsconfig.json`,
                    content: apiConfig.content
                };
                const appFile: ts.projectSystem.File = {
                    path: `${solutionLocation}/app/src/app.ts`,
                    content: apiFile.content
                };
                const sharedConfig: ts.projectSystem.File = {
                    path: `${solutionLocation}/shared/tsconfig.json`,
                    content: JSON.stringify({
                        compilerOptions: {
                            composite: true,
                            outDir: "dist",
                            rootDir: "src",
                        },
                        include: ["src"]
                    })
                };
                const sharedFile: ts.projectSystem.File = {
                    path: `${solutionLocation}/shared/src/index.ts`,
                    content: definition
                };
                const host = ts.projectSystem.createServerHost([ts.projectSystem.libFile, solution, ts.projectSystem.libFile, apiConfig, apiFile, appConfig, appFile, sharedConfig, sharedFile]);
                const session = ts.projectSystem.createSession(host, { logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
                ts.projectSystem.openFilesForSession([apiFile], session);

                // Find all references
                session.executeCommandSeq<ts.projectSystem.protocol.ReferencesRequest>({
                    command: ts.projectSystem.protocol.CommandTypes.References,
                    arguments: ts.projectSystem.protocolFileLocationFromSubstring(apiFile, referenceTerm)
                });

                ts.projectSystem.baselineTsserverLogs("projectReferences", `special handling of localness ${scenario}`, session);
            });
        }

        verify(
            "when using arrow function assignment",
            `export const dog = () => { };`,
            `shared.dog();`,
            "dog"
        );

        verify(
            "when using arrow function as object literal property types",
            `export const foo = { bar: () => { } };`,
            `shared.foo.bar();`,
            "bar"
        );

        verify(
            "when using object literal property",
            `export const foo = {  baz: "BAZ" };`,
            `shared.foo.baz;`,
            "baz"
        );

        verify(
            "when using method of class expression",
            `export const foo = class { fly() {} };`,
            `const instance = new shared.foo();
instance.fly();`,
            "fly"
        );


        verify(
            // when using arrow function as object literal property is loaded through indirect assignment with original declaration local to project is treated as local
            "when using arrow function as object literal property",
            `const local = { bar: () => { } };
export const foo = local;`,
            `shared.foo.bar();`,
            "bar"
        );
    });

    it("when disableSolutionSearching is true, solution and siblings are not loaded", () => {
        const solutionLocation = "/user/username/projects/solution";
        const solution: ts.projectSystem.File = {
            path: `${solutionLocation}/tsconfig.json`,
            content: JSON.stringify({
                files: [],
                include: [],
                references: [
                    { path: "./compiler" },
                    { path: "./services" },
                ]
            })
        };
        const compilerConfig: ts.projectSystem.File = {
            path: `${solutionLocation}/compiler/tsconfig.json`,
            content: JSON.stringify({
                compilerOptions: {
                    composite: true,
                    module: "none",
                    disableSolutionSearching: true
                },
                files: ["./types.ts", "./program.ts"]
            })
        };
        const typesFile: ts.projectSystem.File = {
            path: `${solutionLocation}/compiler/types.ts`,
            content: `
                namespace ts {
                    export interface Program {
                        getSourceFiles(): string[];
                    }
                }`
        };
        const programFile: ts.projectSystem.File = {
            path: `${solutionLocation}/compiler/program.ts`,
            content: `
                namespace ts {
                    export const program: Program = {
                        getSourceFiles: () => [getSourceFile()]
                    };
                    function getSourceFile() { return "something"; }
                }`
        };
        const servicesConfig: ts.projectSystem.File = {
            path: `${solutionLocation}/services/tsconfig.json`,
            content: JSON.stringify({
                compilerOptions: {
                    composite: true
                },
                files: ["./services.ts"],
                references: [
                    { path: "../compiler" }
                ]
            })
        };
        const servicesFile: ts.projectSystem.File = {
            path: `${solutionLocation}/services/services.ts`,
            content: `
                namespace ts {
                    const result = program.getSourceFiles();
                }`
        };

        const files = [ts.projectSystem.libFile, solution, compilerConfig, typesFile, programFile, servicesConfig, servicesFile, ts.projectSystem.libFile];
        const host = ts.projectSystem.createServerHost(files);
        const session = ts.projectSystem.createSession(host, { logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
        ts.projectSystem.openFilesForSession([programFile], session);

        // Find all references
        // No new solutions/projects loaded
        session.executeCommandSeq<ts.projectSystem.protocol.ReferencesRequest>({
            command: ts.projectSystem.protocol.CommandTypes.References,
            arguments: ts.projectSystem.protocolFileLocationFromSubstring(programFile, "getSourceFiles")
        });
        ts.projectSystem.baselineTsserverLogs("projectReferences", `with disableSolutionSearching solution and siblings are not loaded`, session);
    });

    describe("when default project is solution project", () => {
        interface Setup {
            scenario: string;
            solutionOptions?: ts.CompilerOptions;
            solutionFiles?: string[];
            configRefs: string[];
            additionalFiles: readonly ts.projectSystem.File[];
        }
        const main: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/src/main.ts`,
            content: `import { foo } from 'helpers/functions';
export { foo };`
        };
        const helper: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/src/helpers/functions.ts`,
            content: `export const foo = 1;`
        };
        const mainDts: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/target/src/main.d.ts`,
            content: `import { foo } from 'helpers/functions';
export { foo };
//# sourceMappingURL=main.d.ts.map`
        };
        const mainDtsMap: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/target/src/main.d.ts.map`,
            content: `{"version":3,"file":"main.d.ts","sourceRoot":"","sources":["../../src/main.ts"],"names":[],"mappings":"AAAA,OAAO,EAAE,GAAG,EAAE,MAAM,mBAAmB,CAAC;AAExC,OAAO,EAAC,GAAG,EAAC,CAAC"}`
        };
        const helperDts: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/target/src/helpers/functions.d.ts`,
            content: `export declare const foo = 1;
//# sourceMappingURL=functions.d.ts.map`
        };
        const helperDtsMap: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/target/src/helpers/functions.d.ts.map`,
            content: `{"version":3,"file":"functions.d.ts","sourceRoot":"","sources":["../../../src/helpers/functions.ts"],"names":[],"mappings":"AAAA,eAAO,MAAM,GAAG,IAAI,CAAC"}`
        };
        const tsconfigIndirect3: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/indirect3/tsconfig.json`,
            content: JSON.stringify({
                compilerOptions: {
                    baseUrl: "../target/src/"
                },
            })
        };
        const fileResolvingToMainDts: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/indirect3/main.ts`,
            content: `import { foo } from 'main';
foo;
export function bar() {}`
        };
        const tsconfigSrcPath = `${ts.tscWatch.projectRoot}/tsconfig-src.json`;
        const tsconfigPath = `${ts.tscWatch.projectRoot}/tsconfig.json`;
        const dummyFilePath = "/dummy/dummy.ts";
        function setup({ solutionFiles, solutionOptions, configRefs, additionalFiles }: Setup) {
            const tsconfigSrc: ts.projectSystem.File = {
                path: tsconfigSrcPath,
                content: JSON.stringify({
                    compilerOptions: {
                        composite: true,
                        outDir: "./target/",
                        baseUrl: "./src/"
                    },
                    include: ["./src/**/*"]
                })
            };
            const tsconfig: ts.projectSystem.File = {
                path: tsconfigPath,
                content: JSON.stringify({
                    ... (solutionOptions ? { compilerOptions: solutionOptions } : {}),
                    references: configRefs.map(path => ({ path })),
                    files: solutionFiles || []
                })
            };
            const dummyFile: ts.projectSystem.File = {
                path: dummyFilePath,
                content: "let a = 10;"
            };
            const host = ts.projectSystem.createServerHost([
                tsconfigSrc, tsconfig, main, helper,
                ts.projectSystem.libFile, dummyFile,
                mainDts, mainDtsMap, helperDts, helperDtsMap,
                tsconfigIndirect3, fileResolvingToMainDts,
                ...additionalFiles]);
            const session = ts.projectSystem.createSession(host, { canUseEvents: true, logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
            const service = session.getProjectService();
            service.openClientFile(main.path);
            return { session, service, host };
        }

        function verifySolutionScenario(input: Setup) {
            const { session, service, host } = setup(input);

            const info = service.getScriptInfoForPath(main.path as ts.Path)!;
            session.logger.startGroup();
            session.logger.info(`getDefaultProject for ${main.path}: ${info.getDefaultProject().projectName}`);
            session.logger.info(`findDefaultConfiguredProject for ${main.path}: ${service.findDefaultConfiguredProject(info)!.projectName}`);
            session.logger.endGroup();

            // Verify errors
            ts.projectSystem.verifyGetErrRequest({ session, host, files: [main] });

            // Verify collection of script infos
            service.openClientFile(dummyFilePath);

            service.closeClientFile(main.path);
            service.closeClientFile(dummyFilePath);
            service.openClientFile(dummyFilePath);

            service.openClientFile(main.path);
            service.closeClientFile(dummyFilePath);
            service.openClientFile(dummyFilePath);

            // Verify Reload projects
            service.reloadProjects();

            // Find all refs
            session.executeCommandSeq<ts.projectSystem.protocol.ReferencesRequest>({
                command: ts.projectSystem.protocol.CommandTypes.References,
                arguments: ts.projectSystem.protocolFileLocationFromSubstring(main, "foo", { index: 1 })
            }).response as ts.projectSystem.protocol.ReferencesResponseBody;

            service.closeClientFile(main.path);
            service.closeClientFile(dummyFilePath);

            // Verify when declaration map references the file
            service.openClientFile(fileResolvingToMainDts.path);

            // Find all refs from dts include
            session.executeCommandSeq<ts.projectSystem.protocol.ReferencesRequest>({
                command: ts.projectSystem.protocol.CommandTypes.References,
                arguments: ts.projectSystem.protocolFileLocationFromSubstring(fileResolvingToMainDts, "foo")
            }).response as ts.projectSystem.protocol.ReferencesResponseBody;
            ts.projectSystem.baselineTsserverLogs("projectReferences", input.scenario, session);
        }

        function getIndirectProject(postfix: string, optionsToExtend?: ts.CompilerOptions) {
            const tsconfigIndirect: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/tsconfig-indirect${postfix}.json`,
                content: JSON.stringify({
                    compilerOptions: {
                        composite: true,
                        outDir: "./target/",
                        baseUrl: "./src/",
                        ...optionsToExtend
                    },
                    files: [`./indirect${postfix}/main.ts`],
                    references: [{ path: "./tsconfig-src.json" }]
                })
            };
            const indirect: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/indirect${postfix}/main.ts`,
                content: fileResolvingToMainDts.content
            };
            return { tsconfigIndirect, indirect };
        }

        function verifyDisableReferencedProjectLoad(input: Setup) {
            const { session, service } = setup(input);

            const info = service.getScriptInfoForPath(main.path as ts.Path)!;
            session.logger.startGroup();
            session.logger.info(`getDefaultProject for ${main.path}: ${info.getDefaultProject().projectName}`);
            session.logger.info(`findDefaultConfiguredProject for ${main.path}: ${service.findDefaultConfiguredProject(info)?.projectName}`);
            session.logger.endGroup();

            // Verify collection of script infos
            service.openClientFile(dummyFilePath);

            service.closeClientFile(main.path);
            service.closeClientFile(dummyFilePath);
            service.openClientFile(dummyFilePath);

            service.openClientFile(main.path);

            // Verify Reload projects
            service.reloadProjects();
            ts.projectSystem.baselineTsserverLogs("projectReferences", input.scenario, session);
        }

        it("when project is directly referenced by solution", () => {
            verifySolutionScenario({
                scenario: "project is directly referenced by solution",
                configRefs: ["./tsconfig-src.json"],
                additionalFiles: ts.emptyArray,
            });
        });

        it("when project is indirectly referenced by solution", () => {
            const { tsconfigIndirect, indirect } = getIndirectProject("1");
            const { tsconfigIndirect: tsconfigIndirect2, indirect: indirect2 } = getIndirectProject("2");
            verifySolutionScenario({
                scenario: "project is indirectly referenced by solution",
                configRefs: ["./tsconfig-indirect1.json", "./tsconfig-indirect2.json"],
                additionalFiles: [tsconfigIndirect, indirect, tsconfigIndirect2, indirect2],
            });
        });

        it("disables looking into the child project if disableReferencedProjectLoad is set", () => {
            verifyDisableReferencedProjectLoad({
                scenario: "disables looking into the child project if disableReferencedProjectLoad is set",
                solutionOptions: { disableReferencedProjectLoad: true },
                configRefs: ["./tsconfig-src.json"],
                additionalFiles: ts.emptyArray,
            });
        });

        it("disables looking into the child project if disableReferencedProjectLoad is set in indirect project", () => {
            const { tsconfigIndirect, indirect } = getIndirectProject("1", { disableReferencedProjectLoad: true });
            verifyDisableReferencedProjectLoad({
                scenario: "disables looking into the child project if disableReferencedProjectLoad is set in indirect project",
                configRefs: ["./tsconfig-indirect1.json"],
                additionalFiles: [tsconfigIndirect, indirect],
            });
        });

        it("disables looking into the child project if disableReferencedProjectLoad is set in first indirect project but not in another one", () => {
            const { tsconfigIndirect, indirect } = getIndirectProject("1", { disableReferencedProjectLoad: true });
            const { tsconfigIndirect: tsconfigIndirect2, indirect: indirect2 } = getIndirectProject("2");
            verifyDisableReferencedProjectLoad({
                scenario: "disables looking into the child project if disableReferencedProjectLoad is set in first indirect project but not in another one",
                configRefs: ["./tsconfig-indirect1.json", "./tsconfig-indirect2.json"],
                additionalFiles: [tsconfigIndirect, indirect, tsconfigIndirect2, indirect2],
            });
        });

        describe("when solution is project that contains its own files", () => {
            it("when the project found is not solution but references open file through project reference", () => {
                const ownMain: ts.projectSystem.File = {
                    path: `${ts.tscWatch.projectRoot}/own/main.ts`,
                    content: fileResolvingToMainDts.content
                };
                verifySolutionScenario({
                    scenario: "solution with its own files and project found is not solution but references open file through project reference",
                    solutionFiles: [`./own/main.ts`],
                    solutionOptions: {
                        outDir: "./target/",
                        baseUrl: "./src/"
                    },
                    configRefs: ["./tsconfig-src.json"],
                    additionalFiles: [ownMain],
                });
            });

            it("when project is indirectly referenced by solution", () => {
                const ownMain: ts.projectSystem.File = {
                    path: `${ts.tscWatch.projectRoot}/own/main.ts`,
                    content: `import { bar } from 'main';
bar;`
                };
                const { tsconfigIndirect, indirect } = getIndirectProject("1");
                const { tsconfigIndirect: tsconfigIndirect2, indirect: indirect2 } = getIndirectProject("2");
                verifySolutionScenario({
                    scenario: "solution with its own files and project is indirectly referenced by solution",
                    solutionFiles: [`./own/main.ts`],
                    solutionOptions: {
                        outDir: "./target/",
                        baseUrl: "./indirect1/"
                    },
                    configRefs: ["./tsconfig-indirect1.json", "./tsconfig-indirect2.json"],
                    additionalFiles: [tsconfigIndirect, indirect, tsconfigIndirect2, indirect2, ownMain],
                });
            });

            it("disables looking into the child project if disableReferencedProjectLoad is set", () => {
                const ownMain: ts.projectSystem.File = {
                    path: `${ts.tscWatch.projectRoot}/own/main.ts`,
                    content: fileResolvingToMainDts.content
                };
                verifyDisableReferencedProjectLoad({
                    scenario: "solution with its own files and disables looking into the child project if disableReferencedProjectLoad is set",
                    solutionFiles: [`./own/main.ts`],
                    solutionOptions: {
                        outDir: "./target/",
                        baseUrl: "./src/",
                        disableReferencedProjectLoad: true
                    },
                    configRefs: ["./tsconfig-src.json"],
                    additionalFiles: [ownMain],
                });
            });

            it("disables looking into the child project if disableReferencedProjectLoad is set in indirect project", () => {
                const ownMain: ts.projectSystem.File = {
                    path: `${ts.tscWatch.projectRoot}/own/main.ts`,
                    content: `import { bar } from 'main';
bar;`
                };
                const { tsconfigIndirect, indirect } = getIndirectProject("1", { disableReferencedProjectLoad: true });
                verifyDisableReferencedProjectLoad({
                    scenario: "solution with its own files and disables looking into the child project if disableReferencedProjectLoad is set in indirect project",
                    solutionFiles: [`./own/main.ts`],
                    solutionOptions: {
                        outDir: "./target/",
                        baseUrl: "./indirect1/",
                    },
                    configRefs: ["./tsconfig-indirect1.json"],
                    additionalFiles: [tsconfigIndirect, indirect, ownMain],
                });
            });

            it("disables looking into the child project if disableReferencedProjectLoad is set in first indirect project but not in another one", () => {
                const ownMain: ts.projectSystem.File = {
                    path: `${ts.tscWatch.projectRoot}/own/main.ts`,
                    content: `import { bar } from 'main';
bar;`
                };
                const { tsconfigIndirect, indirect } = getIndirectProject("1", { disableReferencedProjectLoad: true });
                const { tsconfigIndirect: tsconfigIndirect2, indirect: indirect2 } = getIndirectProject("2");
                verifyDisableReferencedProjectLoad({
                    scenario: "solution with its own files and disables looking into the child project if disableReferencedProjectLoad is set in first indirect project but not in another one",
                    solutionFiles: [`./own/main.ts`],
                    solutionOptions: {
                        outDir: "./target/",
                        baseUrl: "./indirect1/",
                    },
                    configRefs: ["./tsconfig-indirect1.json", "./tsconfig-indirect2.json"],
                    additionalFiles: [tsconfigIndirect, indirect, tsconfigIndirect2, indirect2, ownMain],
                });
            });
        });
    });

    describe("when new file is added to the referenced project", () => {
        function setup(extendOptionsProject2?: ts.CompilerOptions) {
            const config1: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/projects/project1/tsconfig.json`,
                content: JSON.stringify({
                    compilerOptions: {
                        module: "none",
                        composite: true
                    },
                    exclude: ["temp"]
                })
            };
            const class1: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/projects/project1/class1.ts`,
                content: `class class1 {}`
            };
            const class1Dts: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/projects/project1/class1.d.ts`,
                content: `declare class class1 {}`
            };
            const config2: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/projects/project2/tsconfig.json`,
                content: JSON.stringify({
                    compilerOptions: {
                        module: "none",
                        composite: true,
                        ...(extendOptionsProject2 || {})
                    },
                    references: [
                        { path: "../project1" }
                    ]
                })
            };
            const class2: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/projects/project2/class2.ts`,
                content: `class class2 {}`
            };
            const host = ts.projectSystem.createServerHost([config1, class1, class1Dts, config2, class2, ts.projectSystem.libFile]);
            const session = ts.projectSystem.createSession(host, { logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
            ts.projectSystem.openFilesForSession([class2], session);
            return { host, session, class1 };
        }

        it("when referenced project is not open", () => {
            const { host, session } = setup();

            // Add new class to referenced project
            const class3 = `${ts.tscWatch.projectRoot}/projects/project1/class3.ts`;
            host.writeFile(class3, `class class3 {}`);
            host.checkTimeoutQueueLengthAndRun(2);

            // Add excluded file to referenced project
            host.ensureFileOrFolder({ path: `${ts.tscWatch.projectRoot}/projects/project1/temp/file.d.ts`, content: `declare class file {}` });
            host.checkTimeoutQueueLengthAndRun(0);

            // Add output from new class to referenced project
            const class3Dts = `${ts.tscWatch.projectRoot}/projects/project1/class3.d.ts`;
            host.writeFile(class3Dts, `declare class class3 {}`);
            host.checkTimeoutQueueLengthAndRun(0);
            ts.projectSystem.baselineTsserverLogs("projectReferences", `new file is added to the referenced project when referenced project is not open`, session);
        });

        it("when referenced project is open", () => {
            const { host, session, class1 } = setup();
            ts.projectSystem.openFilesForSession([class1], session);

            // Add new class to referenced project
            const class3 = `${ts.tscWatch.projectRoot}/projects/project1/class3.ts`;
            host.writeFile(class3, `class class3 {}`);
            host.checkTimeoutQueueLengthAndRun(3);
            // Add excluded file to referenced project
            host.ensureFileOrFolder({ path: `${ts.tscWatch.projectRoot}/projects/project1/temp/file.d.ts`, content: `declare class file {}` });
            host.checkTimeoutQueueLengthAndRun(0);
            // Add output from new class to referenced project
            const class3Dts = `${ts.tscWatch.projectRoot}/projects/project1/class3.d.ts`;
            host.writeFile(class3Dts, `declare class class3 {}`);
            host.checkTimeoutQueueLengthAndRun(0);
            ts.projectSystem.baselineTsserverLogs("projectReferences", `new file is added to the referenced project when referenced project is open`, session);
        });

        it("when referenced project is not open with disableSourceOfProjectReferenceRedirect", () => {
            const { host, session } = setup({ disableSourceOfProjectReferenceRedirect: true });

            // Add new class to referenced project
            const class3 = `${ts.tscWatch.projectRoot}/projects/project1/class3.ts`;
            host.writeFile(class3, `class class3 {}`);
            host.checkTimeoutQueueLengthAndRun(2);
            // Add output of new class to referenced project
            const class3Dts = `${ts.tscWatch.projectRoot}/projects/project1/class3.d.ts`;
            host.writeFile(class3Dts, `declare class class3 {}`);
            host.checkTimeoutQueueLengthAndRun(2);
            // Add excluded file to referenced project
            host.ensureFileOrFolder({ path: `${ts.tscWatch.projectRoot}/projects/project1/temp/file.d.ts`, content: `declare class file {}` });
            host.checkTimeoutQueueLengthAndRun(0);
            // Delete output from new class to referenced project
            host.deleteFile(class3Dts);
            host.checkTimeoutQueueLengthAndRun(2);
            // Write back output of new class to referenced project
            host.writeFile(class3Dts, `declare class class3 {}`);
            host.checkTimeoutQueueLengthAndRun(2);
            ts.projectSystem.baselineTsserverLogs("projectReferences", `new file is added to the referenced project when referenced project is not open with disableSourceOfProjectReferenceRedirect`, session);
        });

        it("when referenced project is open with disableSourceOfProjectReferenceRedirect", () => {
            const { host, session, class1 } = setup({ disableSourceOfProjectReferenceRedirect: true });
            ts.projectSystem.openFilesForSession([class1], session);

            // Add new class to referenced project
            const class3 = `${ts.tscWatch.projectRoot}/projects/project1/class3.ts`;
            host.writeFile(class3, `class class3 {}`);
            host.checkTimeoutQueueLengthAndRun(3);
            // Add output of new class to referenced project
            const class3Dts = `${ts.tscWatch.projectRoot}/projects/project1/class3.d.ts`;
            host.writeFile(class3Dts, `declare class class3 {}`);
            host.checkTimeoutQueueLengthAndRun(2);
            // Add excluded file to referenced project
            host.ensureFileOrFolder({ path: `${ts.tscWatch.projectRoot}/projects/project1/temp/file.d.ts`, content: `declare class file {}` });
            host.checkTimeoutQueueLengthAndRun(0);
            // Delete output from new class to referenced project
            host.deleteFile(class3Dts);
            host.checkTimeoutQueueLengthAndRun(2);
            // Write back output of new class to referenced project
            host.writeFile(class3Dts, `declare class class3 {}`);
            host.checkTimeoutQueueLengthAndRun(2);
            ts.projectSystem.baselineTsserverLogs("projectReferences", `new file is added to the referenced project when referenced project is open with disableSourceOfProjectReferenceRedirect`, session);
        });
    });

    describe("auto import with referenced project", () => {
        function verifyAutoImport(built: boolean, disableSourceOfProjectReferenceRedirect?: boolean) {
            const solnConfig: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/tsconfig.json`,
                content: JSON.stringify({
                    files: [],
                    references: [
                        { path: "shared/src/library" },
                        { path: "app/src/program" }
                    ]
                })
            };
            const sharedConfig: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/shared/src/library/tsconfig.json`,
                content: JSON.stringify({
                    compilerOptions: {
                        composite: true,
                        outDir: "../../bld/library"
                    }
                })
            };
            const sharedIndex: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/shared/src/library/index.ts`,
                content: `export function foo() {}`
            };
            const sharedPackage: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/shared/package.json`,
                content: JSON.stringify({
                    name: "shared",
                    version: "1.0.0",
                    main: "bld/library/index.js",
                    types: "bld/library/index.d.ts"
                })
            };
            const appConfig: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/app/src/program/tsconfig.json`,
                content: JSON.stringify({
                    compilerOptions: {
                        composite: true,
                        outDir: "../../bld/program",
                        disableSourceOfProjectReferenceRedirect
                    },
                    references: [
                        { path: "../../../shared/src/library" }
                    ]
                })
            };
            const appBar: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/app/src/program/bar.ts`,
                content: `import {foo} from "shared";`
            };
            const appIndex: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/app/src/program/index.ts`,
                content: `foo`
            };
            const sharedSymlink: ts.projectSystem.SymLink = {
                path: `${ts.tscWatch.projectRoot}/node_modules/shared`,
                symLink: `${ts.tscWatch.projectRoot}/shared`
            };
            const files = [solnConfig, sharedConfig, sharedIndex, sharedPackage, appConfig, appBar, appIndex, sharedSymlink, ts.projectSystem.libFile];
            const host = ts.projectSystem.createServerHost(files);
            if (built) {
                ts.tscWatch.solutionBuildWithBaseline(host, [solnConfig.path]);
                host.clearOutput();
            }
            const session = ts.projectSystem.createSession(host, { logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
            ts.projectSystem.openFilesForSession([appIndex], session);
            session.executeCommandSeq<ts.projectSystem.protocol.CodeFixRequest>({
                command: ts.projectSystem.protocol.CommandTypes.GetCodeFixes,
                arguments: {
                    file: appIndex.path,
                    startLine: 1,
                    startOffset: 1,
                    endLine: 1,
                    endOffset: 4,
                    errorCodes: [ts.Diagnostics.Cannot_find_name_0.code],
                }
            });
            ts.projectSystem.baselineTsserverLogs("projectReferences", `auto import with referenced project${built ? " when built" : ""}${disableSourceOfProjectReferenceRedirect ? " with disableSourceOfProjectReferenceRedirect": ""}`, session);
        }

        it("when project is built", () => {
            verifyAutoImport(/*built*/ true);
        });
        it("when project is not built", () => {
            verifyAutoImport(/*built*/ false);
        });
        it("when disableSourceOfProjectReferenceRedirect is true", () => {
            verifyAutoImport(/*built*/ true, /*disableSourceOfProjectReferenceRedirect*/ true);
        });
    });

    it("when files from two projects are open and one project references", () => {
        function getPackageAndFile(packageName: string, references?: string[], optionsToExtend?: ts.CompilerOptions): [file: ts.projectSystem.File, config: ts.projectSystem.File] {
            const file: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/${packageName}/src/file1.ts`,
                content: `export const ${packageName}Const = 10;`
            };
            const config: ts.projectSystem.File = {
                path: `${ts.tscWatch.projectRoot}/${packageName}/tsconfig.json`,
                content: JSON.stringify({
                    compilerOptions: { composite: true, ...optionsToExtend || {} },
                    references: references?.map(path => ({ path: `../${path}` }))
                })
            };
            return [file, config];
        }
        const [mainFile, mainConfig] = getPackageAndFile("main", ["core", "indirect", "noCoreRef1", "indirectDisabledChildLoad1", "indirectDisabledChildLoad2", "refToCoreRef3", "indirectNoCoreRef"]);
        const [coreFile, coreConfig] = getPackageAndFile("core");
        const [noCoreRef1File, noCoreRef1Config] = getPackageAndFile("noCoreRef1");
        const [indirectFile, indirectConfig] = getPackageAndFile("indirect", ["coreRef1"]);
        const [coreRef1File, coreRef1Config] = getPackageAndFile("coreRef1", ["core"]);
        const [indirectDisabledChildLoad1File, indirectDisabledChildLoad1Config] = getPackageAndFile("indirectDisabledChildLoad1", ["coreRef2"], { disableReferencedProjectLoad: true });
        const [coreRef2File, coreRef2Config] = getPackageAndFile("coreRef2", ["core"]);
        const [indirectDisabledChildLoad2File, indirectDisabledChildLoad2Config] = getPackageAndFile("indirectDisabledChildLoad2", ["coreRef3"], { disableReferencedProjectLoad: true });
        const [coreRef3File, coreRef3Config] = getPackageAndFile("coreRef3", ["core"]);
        const [refToCoreRef3File, refToCoreRef3Config] = getPackageAndFile("refToCoreRef3", ["coreRef3"]);
        const [indirectNoCoreRefFile, indirectNoCoreRefConfig] = getPackageAndFile("indirectNoCoreRef", ["noCoreRef2"]);
        const [noCoreRef2File, noCoreRef2Config] = getPackageAndFile("noCoreRef2");

        const host = ts.projectSystem.createServerHost([
            ts.projectSystem.libFile, mainFile, mainConfig, coreFile, coreConfig, noCoreRef1File, noCoreRef1Config,
            indirectFile, indirectConfig, coreRef1File, coreRef1Config,
            indirectDisabledChildLoad1File, indirectDisabledChildLoad1Config, coreRef2File, coreRef2Config,
            indirectDisabledChildLoad2File, indirectDisabledChildLoad2Config, coreRef3File, coreRef3Config,
            refToCoreRef3File, refToCoreRef3Config,
            indirectNoCoreRefFile, indirectNoCoreRefConfig, noCoreRef2File, noCoreRef2Config
        ], { useCaseSensitiveFileNames: true });
        const session = ts.projectSystem.createSession(host, { logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
        ts.projectSystem.openFilesForSession([mainFile, coreFile], session);

        // Find all refs in coreFile
        session.executeCommandSeq<ts.projectSystem.protocol.ReferencesRequest>({
            command: ts.projectSystem.protocol.CommandTypes.References,
            arguments: ts.projectSystem.protocolFileLocationFromSubstring(coreFile, `coreConst`)
        });
        ts.projectSystem.baselineTsserverLogs("projectReferences", `when files from two projects are open and one project references`, session);
    });

    describe("find refs to decl in other proj", () => {
        const indexA: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/a/index.ts`,
            content: `import { B } from "../b/lib";

const b: B = new B();`
        };

        const configB: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/b/tsconfig.json`,
            content: `{
"compilerOptions": {
    "declarationMap": true,
    "outDir": "lib",
    "composite": true
}
}`
        };

        const indexB: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/b/index.ts`,
            content: `export class B {
    M() {}
}`
        };

        const helperB: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/b/helper.ts`,
            content: `import { B } from ".";

const b: B = new B();`
        };

        const dtsB: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/b/lib/index.d.ts`,
            content: `export declare class B {
    M(): void;
}
//# sourceMappingURL=index.d.ts.map`
        };

        const dtsMapB: ts.projectSystem.File = {
            path: `${ts.tscWatch.projectRoot}/b/lib/index.d.ts.map`,
            content: `{"version":3,"file":"index.d.ts","sourceRoot":"","sources":["../index.ts"],"names":[],"mappings":"AAAA,qBAAa,CAAC;IACV,CAAC;CACJ"}`
        };

        function baselineDisableReferencedProjectLoad(
            projectAlreadyLoaded: boolean,
            disableReferencedProjectLoad: boolean,
            disableSourceOfProjectReferenceRedirect: boolean,
            dtsMapPresent: boolean) {

            // Mangled to stay under windows path length limit
            const subScenario =
                `when proj ${projectAlreadyLoaded ? "is" : "is not"} loaded` +
                ` and refd proj loading is ${disableReferencedProjectLoad ? "disabled" : "enabled"}` +
                ` and proj ref redirects are ${disableSourceOfProjectReferenceRedirect ? "disabled" : "enabled"}` +
                ` and a decl map is ${dtsMapPresent ? "present" : "missing"}`;
            const compilerOptions: ts.CompilerOptions = {
                disableReferencedProjectLoad,
                disableSourceOfProjectReferenceRedirect,
                composite: true
            };

            it(subScenario, () => {
                const configA: ts.projectSystem.File = {
                    path: `${ts.tscWatch.projectRoot}/a/tsconfig.json`,
                    content: `{
        "compilerOptions": ${JSON.stringify(compilerOptions)},
        "references": [{ "path": "../b" }]
    }`
                };

                const host = ts.projectSystem.createServerHost([configA, indexA, configB, indexB, helperB, dtsB, ...(dtsMapPresent ? [dtsMapB] : [])]);
                const session = ts.projectSystem.createSession(host, { logger: ts.projectSystem.createLoggerWithInMemoryLogs(host) });
                ts.projectSystem.openFilesForSession([indexA, ...(projectAlreadyLoaded ? [helperB] : [])], session);

                session.executeCommandSeq<ts.projectSystem.protocol.ReferencesRequest>({
                    command: ts.projectSystem.protocol.CommandTypes.References,
                    arguments: ts.projectSystem.protocolFileLocationFromSubstring(indexA, `B`, { index: 1 })
                });
                ts.projectSystem.baselineTsserverLogs("projectReferences", `find refs to decl in other proj ${subScenario}`, session);
            });
        }

        /* eslint-disable local/boolean-trivia */

        // Pre-loaded = A file from project B is already open when FAR is invoked
        // dRPL = Project A has disableReferencedProjectLoad
        // dSOPRR = Project A has disableSourceOfProjectReferenceRedirect
        // Map = The declaration map file b/lib/index.d.ts.map exists
        // B refs = files under directory b in which references are found (all scenarios find all references in a/index.ts)

        //                                   Pre-loaded | dRPL   | dSOPRR | Map      | B state    | Notes        | B refs              | Notes
        //                                   -----------+--------+--------+----------+------------+--------------+---------------------+---------------------------------------------------
        baselineDisableReferencedProjectLoad(true,        true,    true,    true);  // Pre-loaded |              | index.ts, helper.ts | Via map and pre-loaded project
        baselineDisableReferencedProjectLoad(true,        true,    true,    false); // Pre-loaded |              | lib/index.d.ts      | Even though project is loaded
        baselineDisableReferencedProjectLoad(true,        true,    false,   true);  // Pre-loaded |              | index.ts, helper.ts |
        baselineDisableReferencedProjectLoad(true,        true,    false,   false); // Pre-loaded |              | index.ts, helper.ts |
        baselineDisableReferencedProjectLoad(true,        false,   true,    true);  // Pre-loaded |              | index.ts, helper.ts | Via map and pre-loaded project
        baselineDisableReferencedProjectLoad(true,        false,   true,    false); // Pre-loaded |              | lib/index.d.ts      | Even though project is loaded
        baselineDisableReferencedProjectLoad(true,        false,   false,   true);  // Pre-loaded |              | index.ts, helper.ts |
        baselineDisableReferencedProjectLoad(true,        false,   false,   false); // Pre-loaded |              | index.ts, helper.ts |
        baselineDisableReferencedProjectLoad(false,       true,    true,    true);  // Not loaded |              | lib/index.d.ts      | Even though map is present
        baselineDisableReferencedProjectLoad(false,       true,    true,    false); // Not loaded |              | lib/index.d.ts      |
        baselineDisableReferencedProjectLoad(false,       true,    false,   true);  // Not loaded |              | index.ts            | But not helper.ts, which is not referenced from a
        baselineDisableReferencedProjectLoad(false,       true,    false,   false); // Not loaded |              | index.ts            | But not helper.ts, which is not referenced from a
        baselineDisableReferencedProjectLoad(false,       false,   true,    true);  // Loaded     | Via map      | index.ts, helper.ts | Via map and newly loaded project
        baselineDisableReferencedProjectLoad(false,       false,   true,    false); // Not loaded |              | lib/index.d.ts      |
        baselineDisableReferencedProjectLoad(false,       false,   false,   true);  // Loaded     | Via redirect | index.ts, helper.ts |
        baselineDisableReferencedProjectLoad(false,       false,   false,   false); // Loaded     | Via redirect | index.ts, helper.ts |

        /* eslint-enable local/boolean-trivia */
    });
});
