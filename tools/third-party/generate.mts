import assert from "node:assert/strict";
import fs from "node:fs";
import { fileURLToPath } from "node:url";

export interface License {
    source: string;
    text: string;
}

export interface Dependency {
    type: "npm" | "go" | "git";
    name: string;
    version: string;
    tag?: string;
    licenses: License[];
    licensesConcluded?: string[];
}

export const directory = fileURLToPath(new URL("./", import.meta.url));
export const json = (value: unknown) => JSON.stringify(value, undefined, 4) + "\n";
const normalize = (text: string) => text.replaceAll("\r\n", "\n").trim();

export function merge(dependencies: Dependency[]): Dependency[] {
    const merged = new Map<string, Dependency>();
    for (const dependency of dependencies) {
        assert(dependency.name && dependency.version, "Dependency needs a name and version");
        assert(dependency.licenses.length && dependency.licenses.every(file => file.source && normalize(file.text)), `Missing license text for ${dependency.name}`);
        const key = `${dependency.type}:${dependency.name}@${dependency.version}`;
        const previous = merged.get(key);
        assert(!previous?.tag || !dependency.tag || previous.tag === dependency.tag, `Conflicting Git tags for ${key}`);
        const licenses = [...(previous?.licenses ?? []), ...dependency.licenses];
        const files = new Map(licenses.map(file => [file.source, normalize(file.text)]));
        assert(licenses.every(file => files.get(file.source) === normalize(file.text)), `Conflicting license texts for ${key}`);
        merged.set(key, {
            ...dependency,
            tag: dependency.tag ?? previous?.tag,
            licenses: [...files.keys()].sort().map(source => ({ source, text: files.get(source)! })),
            licensesConcluded: [...new Set([...(previous?.licensesConcluded ?? []), ...(dependency.licensesConcluded ?? [])])].sort(),
        });
    }
    return [...merged.keys()].sort().map(key => merged.get(key)!);
}

function render(dependencies: Dependency[], header: string) {
    const entries = merge(dependencies);
    assert(entries.length, "Empty dependency inventory");
    for (const entry of entries) {
        assert(entry.licensesConcluded?.length && entry.licensesConcluded.every(expression => expression.trim()), `Missing SPDX licenses for ${entry.name}; run update:third-party`);
    }
    const sorted = entries.map(entry => ({
        entry,
        name: entry.type === "git" ? new URL(entry.name).pathname.slice(1).replace(/\.git$/, "") : entry.name,
    })).sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()));
    const separator = "---------------------------------------------------------";
    let notice = normalize(header) + "\n\n";
    for (const { entry, name } of sorted) {
        const url = entry.type === "git" ? entry.name
            : entry.type === "npm" ? `https://www.npmjs.com/package/${entry.name}/v/${entry.version}`
            : `https://pkg.go.dev/${entry.name}@${entry.version}`;
        const text = [...new Set(entry.licenses.map(file => file.text))].join("\n\n");
        notice += `\n${separator}\n\n${name} ${entry.version} - ${entry.licensesConcluded!.join(" AND ")}\n${url}\n\n${text}\n`;
    }
    notice += `\n${separator}\n`;
    return {
        manifest: json({
            $schema: "https://json.schemastore.org/component-detection-manifest.json",
            version: 1,
            registrations: entries.map(({ type, name, version, tag, licensesConcluded }) => ({
                component: {
                    type,
                    [type]: type === "git" ? { repositoryUrl: name, commitHash: version, tag } : { name, version },
                },
                developmentDependency: false,
                licensesConcluded,
            })),
        }),
        notice,
    };
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
    const output = render(
        JSON.parse(fs.readFileSync(`${directory}/generated/dependencies.json`, "utf8")),
        fs.readFileSync(`${directory}/inputs/notice-header.txt`, "utf8"),
    );
    fs.writeFileSync(`${directory}/generated/NOTICE.txt`, output.notice);
    fs.writeFileSync(`${directory}/generated/cgmanifest.json`, output.manifest);
}
