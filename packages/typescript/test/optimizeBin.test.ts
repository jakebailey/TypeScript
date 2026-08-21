import assert from "node:assert";
import fs from "node:fs";
import path from "node:path";
import {
    describe,
    test,
} from "node:test";
import {
    fileURLToPath,
    pathToFileURL,
} from "node:url";

describe("optimizeBin", () => {
    test("replaces the installed npm bin with a symlink to the platform executable", {
        skip: process.platform === "win32",
    }, async () => {
        const testDir = path.dirname(fileURLToPath(import.meta.url));
        const root = fs.mkdtempSync(path.join(testDir, ".optimizeBin-"));
        try {
            const packageDir = path.join(root, "node_modules", "@typescript", "typescript");
            const libDir = path.join(packageDir, "lib");
            const binDir = path.join(packageDir, "bin");
            const exe = path.join(root, "node_modules", "@typescript", `typescript-${process.platform}-${process.arch}`, "lib", "tsc");
            fs.mkdirSync(libDir, { recursive: true });
            fs.mkdirSync(binDir);
            fs.mkdirSync(path.dirname(exe), { recursive: true });
            fs.writeFileSync(path.join(packageDir, "package.json"), '{"type":"module"}\n');
            fs.copyFileSync(new URL("../lib/optimizeBin.js", import.meta.url), path.join(libDir, "optimizeBin.js"));
            fs.writeFileSync(path.join(libDir, "getExePath.js"), `export default () => ${JSON.stringify(exe)};\n`);
            fs.writeFileSync(path.join(binDir, "tsc"), '#!/usr/bin/env node\nimport "../lib/tsc.js";\n');
            fs.writeFileSync(exe, "");

            await import(pathToFileURL(path.join(libDir, "optimizeBin.js")).href);

            const bin = path.join(binDir, "tsc");
            assert(fs.lstatSync(bin).isSymbolicLink());
            assert.strictEqual(fs.realpathSync(bin), fs.realpathSync(exe));
        }
        finally {
            fs.rmSync(root, { recursive: true, force: true });
        }
    });
});
