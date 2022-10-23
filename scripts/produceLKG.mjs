import fs from "fs-extra";
import path from "path";
import glob from "glob";
import url from "url";
import del from "del";

const __filename = url.fileURLToPath(new URL(import.meta.url));
const __dirname = path.dirname(__filename);

const root = path.join(__dirname, "..");
const source = path.join(root, "built/local");
const dest = path.join(root, "lib");
const copyright = fs.readFileSync(path.join(__dirname, "../CopyrightNotice.txt"), "utf-8");

async function produceLKG() {
    console.log(`Building LKG from ${source} to ${dest}`);
    await del(`${dest.replace(/\\/g, "/")}/**`, { ignore: ["**/README.md"] });
    await fs.mkdirp(dest);
    await copyLibFiles();
    await copyLocalizedDiagnostics();
    await copyTypesMap();
    await copyScriptOutputs();
    await copyDeclarationOutputs();
    await writeGitAttributes();
}

async function copyLibFiles() {
    await copyFilesWithGlob("lib?(.*).d.ts");
}

async function copyLocalizedDiagnostics() {
    const dir = await fs.readdir(source);
    const ignoredFolders = ["enu"];

    for (const d of dir) {
        const fileName = path.join(source, d);
        if (
            fs.statSync(fileName).isDirectory() &&
            ignoredFolders.indexOf(d) < 0
        ) {
            await fs.copy(fileName, path.join(dest, d));
        }
    }
}

async function copyTypesMap() {
    await copyFromBuiltLocal("typesMap.json"); // Cannot accommodate copyright header
}

async function copyScriptOutputs() {
    await copyWithCopyright("cancellationToken.js");
    await copyWithCopyright("tsc.release.js", "tsc.js");
    await copyWithCopyright("tsserver.js");
    await copyWithCopyright("dynamicImportCompat.js");
    await copyFromBuiltLocal("tsserverlibrary.js"); // copyright added by build
    await copyFromBuiltLocal("typescript.js"); // copyright added by build
    await copyFromBuiltLocal("typescriptServices.js"); // copyright added by build
    await copyWithCopyright("typingsInstaller.js");
    await copyWithCopyright("watchGuard.js");
}

async function copyDeclarationOutputs() {
    await copyFromBuiltLocal("tsserverlibrary.d.ts"); // copyright added by build
    await copyFromBuiltLocal("typescript.d.ts"); // copyright added by build
    await copyFromBuiltLocal("typescriptServices.d.ts"); // copyright added by build
}

async function writeGitAttributes() {
    await fs.writeFile(path.join(dest, ".gitattributes"), `* text eol=lf`, "utf-8");
}

/**
 * @param {string} fileName
 * @param {string} destName
 */
async function copyWithCopyright(fileName, destName = fileName) {
    const content = await fs.readFile(path.join(source, fileName), "utf-8");
    await fs.writeFile(path.join(dest, destName), copyright + "\n" + content);
}

/**
 * @param {string} fileName
 */
async function copyFromBuiltLocal(fileName) {
    await fs.copy(path.join(source, fileName), path.join(dest, fileName));
}

/**
 * @param {string} pattern
 */
async function copyFilesWithGlob(pattern) {
    const files = glob.sync(pattern, { cwd: source }).map(f => path.basename(f));
    for (const f of files) {
        await copyFromBuiltLocal(f);
    }
    console.log(`Copied ${files.length} files matching pattern ${pattern}`);
}

process.on("unhandledRejection", err => {
    throw err;
});
produceLKG().then(() => console.log("Done"), err => {
    throw err;
});
