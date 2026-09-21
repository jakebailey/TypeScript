import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";
import * as tar from "tar";
import { x } from "tinyexec";
import {
    PACKAGE as unicodePackage,
    UNICODE_VERSION as unicodeVersion,
} from "../../tsc/internal/stringutil/_scripts/generate-unicode-data.mts";
import {
    type Dependency,
    directory,
    json,
    type License,
    merge,
} from "./generate.mts";

interface Lockfile {
    packages: Record<string, {
        name?: string;
        version?: string;
        dev?: boolean;
        link?: boolean;
        resolved?: string;
        integrity?: string;
        dependencies?: Record<string, string>;
        optionalDependencies?: Record<string, string>;
        peerDependencies?: Record<string, string>;
        peerDependenciesMeta?: Record<string, { optional?: boolean; }>;
    }>;
}

interface Inputs {
    domGenerator: string;
    npm?: string[];
    copied: { repositoryUrl: string; commitHash: string; tag?: string; }[];
}

const root = fileURLToPath(new URL("../../", import.meta.url));
const domRepository = "https://github.com/microsoft/TypeScript-DOM-lib-generator";
const readJson = (file: string) => JSON.parse(fs.readFileSync(file, "utf8"));
const packageName = (directory: string, entry: { name?: string; }) => entry.name ?? directory.split("node_modules/").at(-1)!;

function lockedUnicodeVersion(lock: Lockfile): string {
    assert(readJson(path.join(root, "package.json")).devDependencies[unicodePackage], `The Unicode table generator selects ${unicodePackage}, but package.json does not declare it`);
    const version = lock.packages[`node_modules/${unicodePackage}`]?.version;
    assert(version, `Missing locked version of ${unicodePackage}`);
    return version;
}

function checkUnicodeInventory() {
    const version = lockedUnicodeVersion(readJson(path.join(root, "package-lock.json")));
    const inventory: Dependency[] = readJson(`${directory}/generated/dependencies.json`);
    const unicode = inventory.filter(entry => entry.type === "npm" && entry.name.startsWith("@unicode/unicode-"));
    assert(unicode.length === 1 && unicode[0].name === unicodePackage && unicode[0].version === version, `Unicode notices are stale: expected ${unicodePackage}@${version}. Run npx hereby update:third-party`);
}

function npmPackages(lock: Lockfile, roots: string[], additional: string[] = []) {
    const seen = new Set<string>();
    const packages: { name: string; version: string; integrity?: string; }[] = [];
    function resolve(name: string, from: string, optional = false) {
        for (let directory = from;; directory = path.posix.dirname(directory)) {
            const candidate = path.posix.join(directory, "node_modules", name);
            if (lock.packages[candidate]) return visit(candidate);
            if (directory === "." || directory === "") break;
        }
        assert(optional, `Missing locked npm dependency: ${name} from ${from}`);
    }
    function visit(directory: string) {
        if (seen.has(directory)) return;
        seen.add(directory);
        const entry = lock.packages[directory];
        assert(entry, `Missing lockfile entry: ${directory}`);
        if (entry.link) {
            assert(entry.resolved, `Missing workspace target: ${directory}`);
            return visit(entry.resolved);
        }
        assert(entry.version, `Missing version for ${directory}`);
        if (directory.includes("node_modules/")) packages.push({ name: packageName(directory, entry), version: entry.version, integrity: entry.integrity });
        for (const name of Object.keys({ ...entry.dependencies, ...entry.optionalDependencies, ...entry.peerDependencies })) {
            resolve(name, directory, !!entry.optionalDependencies?.[name] || !!entry.peerDependenciesMeta?.[name]?.optional);
        }
    }
    roots.forEach(visit);
    additional.forEach(name => resolve(name, ""));
    return packages;
}

function domPackages(lock: Lockfile) {
    const packages = Object.entries(lock.packages)
        .filter(([directory]) => /^node_modules\/(?:@mdn\/browser-compat-data$|@webref\/[^/]+$)/.test(directory))
        .map(([directory, entry]) => {
            assert(entry.version, `Missing DOM dependency version: ${directory}`);
            return { name: packageName(directory, entry), version: entry.version, integrity: entry.integrity };
        });
    assert(packages.length, "No DOM data dependencies found in the generator lockfile");
    return packages;
}

function isLicense(file: string) {
    return /(?:^|\/)(?:licen[cs]es?|copying|copyright|notice|patents|authors|third[-_]?party[-_]?notices?)(?:[._/-]|$)/i.test(file)
        && !/\.(?:[cm]?[jt]sx?|go|py|map|json|ya?ml)$/i.test(file);
}

function readLicenses(directory: string, source: string, prefix = ""): License[] {
    return fs.readdirSync(directory, { withFileTypes: true }).sort((a, b) => a.name < b.name ? -1 : a.name > b.name ? 1 : 0).flatMap(entry => {
        const relative = prefix + entry.name;
        if (entry.isDirectory() && entry.name !== "node_modules" && entry.name !== ".git") {
            return readLicenses(path.join(directory, entry.name), source, relative + "/");
        }
        return entry.isFile() && isLicense(relative)
            ? [{ source: `${source}#${relative}`, text: fs.readFileSync(path.join(directory, entry.name), "utf8") }]
            : [];
    });
}

async function fetchBytes(url: string) {
    const response = await fetch(url);
    assert(response.ok, `Cannot fetch ${url}: ${response.status} ${response.statusText}`);
    return Buffer.from(await response.arrayBuffer());
}

async function remoteLicense(url: string): Promise<License> {
    return { source: url, text: (await fetchBytes(url)).toString("utf8") };
}

async function repositoryLicenses(repository: string, ref: string): Promise<License[]> {
    const match = repository.match(/github\.com[/:]([^/]+\/[^/#]+?)(?:\.git)?$/);
    assert(match, `Expected a GitHub repository: ${repository}`);
    const files: { type: string; name: string; download_url: string; }[] = JSON.parse(
        (await fetchBytes(`https://api.github.com/repos/${match[1]}/contents?ref=${encodeURIComponent(ref)}`)).toString("utf8"),
    );
    const licenses = files.filter(file => file.type === "file" && isLicense(file.name));
    assert(licenses.length, `No root license documents in ${repository} at ${ref}`);
    return Promise.all(licenses.map(file => remoteLicense(file.download_url)));
}

async function unicodeLicenses(bugs: string): Promise<License[]> {
    assert(bugs.endsWith("/issues"), `Unexpected Unicode generator issue tracker: ${bugs}`);
    const licenses = await repositoryLicenses(bugs.slice(0, -"/issues".length), "HEAD");
    const readme = (await fetchBytes(`https://www.unicode.org/Public/${unicodeVersion}/ucd/ReadMe.txt`)).toString("utf8");
    const termsUrl = readme.match(/https:\/\/(?:www\.)?unicode\.org\/terms_of_use\.html/)?.[0];
    assert(termsUrl, `No terms-of-use link in the Unicode ${unicodeVersion} data README`);
    const terms = (await fetchBytes(termsUrl)).toString("utf8");
    const licenseUrl = terms.match(/href=["']([^"']*\/license\.txt)["']/i)?.[1];
    assert(licenseUrl, `No Unicode license link in ${termsUrl}`);
    return [...licenses, await remoteLicense(new URL(licenseUrl, termsUrl).href)];
}

async function mdnLicenses(domSource: string): Promise<License[]> {
    const snapshot: { mdn_url: string; }[] = JSON.parse((await fetchBytes(`${domSource}/inputfiles/mdn.json`)).toString("utf8"));
    assert(snapshot[0]?.mdn_url, "No MDN source URL in the DOM generator's documentation snapshot");
    const page = new URL(snapshot[0].mdn_url.replace(/\/$/, "") + "/metadata.json", "https://developer.mozilla.org");
    const metadata = JSON.parse((await fetchBytes(page.href)).toString("utf8"));
    const source = metadata.source?.github_url?.match(/^(https:\/\/github\.com\/[^/]+\/[^/]+)\/blob\/([^/]+)\//);
    assert(source, `No GitHub source in ${page.href}`);
    return repositoryLicenses(source[1], source[2]);
}

async function archiveLicenses(file: string, source: string): Promise<License[]> {
    const files: Promise<License>[] = [];
    await tar.t({
        file,
        strict: true,
        onReadEntry(entry) {
            const relative = entry.path.replace(/^[^/]+\//, "");
            if (entry.type !== "File" || !isLicense(relative) || relative.split("/").includes("node_modules")) return;
            files.push((async () => {
                const chunks: Buffer[] = [];
                for await (const chunk of entry) chunks.push(chunk);
                return { source: `${source}#${relative}`, text: Buffer.concat(chunks).toString("utf8") };
            })());
        },
    });
    return Promise.all(files);
}

function goDependencies(targets: string[]): Dependency[] {
    assert(targets.length, "Supply release targets with --target=GOOS/GOARCH");
    const modules = new Map<string, Dependency>();
    for (const target of targets) {
        const [GOOS, GOARCH] = target.split("/");
        assert(GOOS && GOARCH, `Invalid Go target: ${target}`);
        const output = execFileSync("go", [
            "-C",
            "tsc",
            "list",
            "-mod=readonly",
            "-deps",
            "-tags=noembed",
            "-f",
            "{{if .Module}}{{if not .Module.Main}}{{.Module.Path}}\t{{.Module.Version}}\t{{.Module.Dir}}\t{{if .Module.Replace}}replacement{{end}}{{end}}{{end}}",
            "./cmd/tsc",
        ], {
            cwd: root,
            encoding: "utf8",
            env: { ...process.env, GOOS, GOARCH, GOARM: "6", CGO_ENABLED: "0", GOWORK: "off", GOFLAGS: "" },
        });
        for (const line of output.trim().split(/\r?\n/).filter(Boolean)) {
            const [name, version, directory, replacement] = line.split("\t");
            assert(!replacement, `Replaced Go module needs an explicit source: ${name}`);
            assert(name && version && directory, `Incomplete Go module: ${line}`);
            if (!modules.has(name)) {
                modules.set(name, { type: "go", name, version, licenses: readLicenses(directory, `${name}@${version}`) });
            }
        }
    }
    return [...modules.values()];
}

async function scanLicenses(dependencies: Dependency[], work: string) {
    const input = path.join(work, "licenses");
    const output = path.join(work, "scancode.json");
    const documents = new Map<string, { dependency: Dependency; source: string; }>();
    dependencies.forEach((dependency, i) => {
        fs.mkdirSync(path.join(input, String(i)), { recursive: true });
        dependency.licenses.forEach(({ source, text }, j) => {
            const file = `${i}/${j}.txt`;
            fs.writeFileSync(path.join(input, file), text);
            documents.set(file, { dependency, source });
        });
    });
    await x("uvx", [
        "--python",
        "3.12",
        "--from",
        "scancode-toolkit==32.5.0",
        "scancode",
        "--license",
        "--strip-root",
        "--processes",
        "1",
        "--json",
        output,
        input,
    ], { throwOnError: true, nodeOptions: { stdio: "inherit" } });
    const scan: { files: { path: string; detected_license_expression_spdx?: string; scan_errors: string[]; }[]; } = readJson(output);
    for (const file of scan.files) {
        assert(!file.scan_errors.length, `License scan failed for ${file.path}: ${file.scan_errors.join("; ")}`);
        const expression = file.detected_license_expression_spdx;
        if (!expression) continue;
        const document = documents.get(file.path);
        assert(document, `Unexpected scanner path: ${file.path}`);
        assert(!/LicenseRef-scancode-unknown|NOASSERTION/.test(expression), `Unidentified license in ${document.source}: ${expression}`);
        (document.dependency.licensesConcluded ??= []).push(expression);
    }
    for (const dependency of dependencies) {
        dependency.licensesConcluded = [...new Set(dependency.licensesConcluded)].sort();
        assert(dependency.licensesConcluded.length, `No SPDX license detected for ${dependency.name}`);
    }
}

async function update(targets: string[]) {
    const inputs: Inputs = readJson(`${directory}/inputs/dependencies.json`);
    assert(/^[a-f0-9]{40}$/.test(inputs.domGenerator), "DOM generator must be a full commit hash");
    const lock: Lockfile = readJson(path.join(root, "package-lock.json"));
    lockedUnicodeVersion(lock);
    const dependencies = goDependencies(targets);
    const roots = ["packages/typescript"];
    for (const file of fs.globSync("packages/typescript/vendor/**/package.json", { cwd: root })) {
        const metadata = readJson(path.join(root, file));
        const { name, version } = metadata;
        const source = path.dirname(file).split(path.sep).join("/");
        lock.packages[source] = metadata;
        roots.push(source);
        dependencies.push({ type: "npm", name, version, licenses: readLicenses(path.dirname(path.join(root, file)), source) });
    }
    for (const source of inputs.copied) {
        assert(/^[a-f0-9]{40}$/.test(source.commitHash), `Expected a full Git commit hash for ${source.repositoryUrl}`);
        dependencies.push({
            type: "git",
            name: source.repositoryUrl,
            version: source.commitHash,
            tag: source.tag,
            licenses: await repositoryLicenses(source.repositoryUrl, source.commitHash),
        });
    }

    const domSource = `https://raw.githubusercontent.com/microsoft/TypeScript-DOM-lib-generator/${inputs.domGenerator}`;
    const domLock: Lockfile = JSON.parse((await fetchBytes(`${domSource}/package-lock.json`)).toString("utf8"));
    dependencies.push({
        type: "git",
        name: domRepository,
        version: inputs.domGenerator,
        licenses: [await remoteLicense(`${domSource}/LICENSE.txt`), ...await mdnLicenses(domSource)],
    });
    const packages = [...npmPackages(lock, roots, [unicodePackage, ...inputs.npm ?? []]), ...domPackages(domLock)];
    fs.mkdirSync(path.join(root, "built"), { recursive: true });
    const work = fs.mkdtempSync(path.join(root, "built/third-party-update-"));
    try {
        const seen = new Set<string>();
        for (const { name, version, integrity } of packages) {
            const spec = `${name}@${version}`;
            if (seen.has(spec)) continue;
            seen.add(spec);
            console.log(`Collecting ${spec}`);
            const result = await x("npm", ["pack", spec, "--ignore-scripts", "--json", "--pack-destination", work], { throwOnError: true });
            const [packed] = JSON.parse(result.stdout);
            assert.equal(packed.name, name);
            assert.equal(packed.version, version);
            if (integrity) assert.equal(packed.integrity, integrity, `Lockfile integrity mismatch: ${spec}`);
            let licenses = await archiveLicenses(path.join(work, packed.filename), `https://www.npmjs.com/package/${name}/v/${version}`);
            if (name === unicodePackage || !licenses.length) {
                const metadata = JSON.parse((await x("npm", ["view", spec, "--json"], { throwOnError: true })).stdout);
                if (name === unicodePackage) {
                    const bugs = typeof metadata.bugs === "string" ? metadata.bugs : metadata.bugs?.url;
                    assert(bugs, `No Unicode generator provenance in ${spec}`);
                    licenses.push(...await unicodeLicenses(bugs));
                }
                else {
                    const repository = typeof metadata.repository === "string" ? metadata.repository : metadata.repository?.url;
                    assert(repository && /^[a-f0-9]{40}$/.test(metadata.gitHead), `Cannot resolve upstream license documents for ${spec}`);
                    licenses = await repositoryLicenses(repository, metadata.gitHead);
                }
            }
            dependencies.push({ type: "npm", name, version, licenses });
        }
        const inventory = merge(dependencies);
        await scanLicenses(inventory, work);
        fs.mkdirSync(`${directory}/generated`, { recursive: true });
        fs.writeFileSync(`${directory}/generated/dependencies.json`, json(inventory));
        console.log(`Updated ${inventory.length} third-party dependencies.`);
    }
    finally {
        fs.rmSync(work, { recursive: true, force: true });
    }
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
    const { values } = parseArgs({ options: { target: { type: "string", multiple: true }, check: { type: "boolean" } } });
    if (values.check) checkUnicodeInventory();
    else await update(values.target ?? []);
}
