import {
    endsWith,
    equateStringsCaseInsensitive,
    equateStringsCaseSensitive,
    lastOrUndefined,
    some,
    startsWith,
} from "./core";
import {
    CharacterCodes,
    Path,
} from "./types";

/**
 * Internally, we represent paths as strings with '/' as the directory separator.
 * When we make system calls (eg: LanguageServiceHost.getDirectory()),
 * we expect the host to correctly handle paths in our specified format.
 *
 * @internal
 */
const directorySeparator = "/";
/** @internal */
const altDirectorySeparator = "\\";
const urlSchemeSeparator = "://";
const backslashRegExp = /\\/g;

//// Path Tests

/**
 * Determines whether a charCode corresponds to `/` or `\`.
 *
 * @internal
 */
function isAnyDirectorySeparator(charCode: number): boolean {
    return charCode === CharacterCodes.slash || charCode === CharacterCodes.backslash;
}

/** @internal */
export function fileExtensionIs(path: string, extension: string): boolean {
    return path.length > extension.length && endsWith(path, extension);
}

/**
 * Determines whether a path has a trailing separator (`/` or `\\`).
 *
 * @internal
 */
function hasTrailingDirectorySeparator(path: string) {
    return path.length > 0 && isAnyDirectorySeparator(path.charCodeAt(path.length - 1));
}

//// Path Parsing

function isVolumeCharacter(charCode: number) {
    return (charCode >= CharacterCodes.a && charCode <= CharacterCodes.z) ||
        (charCode >= CharacterCodes.A && charCode <= CharacterCodes.Z);
}

function getFileUrlVolumeSeparatorEnd(url: string, start: number) {
    const ch0 = url.charCodeAt(start);
    if (ch0 === CharacterCodes.colon) return start + 1;
    if (ch0 === CharacterCodes.percent && url.charCodeAt(start + 1) === CharacterCodes._3) {
        const ch2 = url.charCodeAt(start + 2);
        if (ch2 === CharacterCodes.a || ch2 === CharacterCodes.A) return start + 3;
    }
    return -1;
}

/**
 * Returns length of the root part of a path or URL (i.e. length of "/", "x:/", "//server/share/, file:///user/files").
 * If the root is part of a URL, the twos-complement of the root length is returned.
 */
function getEncodedRootLength(path: string): number {
    if (!path) return 0;
    const ch0 = path.charCodeAt(0);

    // POSIX or UNC
    if (ch0 === CharacterCodes.slash || ch0 === CharacterCodes.backslash) {
        if (path.charCodeAt(1) !== ch0) return 1; // POSIX: "/" (or non-normalized "\")

        const p1 = path.indexOf(ch0 === CharacterCodes.slash ? directorySeparator : altDirectorySeparator, 2);
        if (p1 < 0) return path.length; // UNC: "//server" or "\\server"

        return p1 + 1; // UNC: "//server/" or "\\server\"
    }

    // DOS
    if (isVolumeCharacter(ch0) && path.charCodeAt(1) === CharacterCodes.colon) {
        const ch2 = path.charCodeAt(2);
        if (ch2 === CharacterCodes.slash || ch2 === CharacterCodes.backslash) return 3; // DOS: "c:/" or "c:\"
        if (path.length === 2) return 2; // DOS: "c:" (but not "c:d")
    }

    // URL
    const schemeEnd = path.indexOf(urlSchemeSeparator);
    if (schemeEnd !== -1) {
        const authorityStart = schemeEnd + urlSchemeSeparator.length;
        const authorityEnd = path.indexOf(directorySeparator, authorityStart);
        if (authorityEnd !== -1) { // URL: "file:///", "file://server/", "file://server/path"
            // For local "file" URLs, include the leading DOS volume (if present).
            // Per https://www.ietf.org/rfc/rfc1738.txt, a host of "" or "localhost" is a
            // special case interpreted as "the machine from which the URL is being interpreted".
            const scheme = path.slice(0, schemeEnd);
            const authority = path.slice(authorityStart, authorityEnd);
            if (
                scheme === "file" && (authority === "" || authority === "localhost") &&
                isVolumeCharacter(path.charCodeAt(authorityEnd + 1))
            ) {
                const volumeSeparatorEnd = getFileUrlVolumeSeparatorEnd(path, authorityEnd + 2);
                if (volumeSeparatorEnd !== -1) {
                    if (path.charCodeAt(volumeSeparatorEnd) === CharacterCodes.slash) {
                        // URL: "file:///c:/", "file://localhost/c:/", "file:///c%3a/", "file://localhost/c%3a/"
                        return ~(volumeSeparatorEnd + 1);
                    }
                    if (volumeSeparatorEnd === path.length) {
                        // URL: "file:///c:", "file://localhost/c:", "file:///c$3a", "file://localhost/c%3a"
                        // but not "file:///c:d" or "file:///c%3ad"
                        return ~volumeSeparatorEnd;
                    }
                }
            }
            return ~(authorityEnd + 1); // URL: "file://server/", "http://server/"
        }
        return ~path.length; // URL: "file://server", "http://server"
    }

    // relative
    return 0;
}

/**
 * Returns length of the root part of a path or URL (i.e. length of "/", "x:/", "//server/share/, file:///user/files").
 *
 * For example:
 * ```ts
 * getRootLength("a") === 0                   // ""
 * getRootLength("/") === 1                   // "/"
 * getRootLength("c:") === 2                  // "c:"
 * getRootLength("c:d") === 0                 // ""
 * getRootLength("c:/") === 3                 // "c:/"
 * getRootLength("c:\\") === 3                // "c:\\"
 * getRootLength("//server") === 7            // "//server"
 * getRootLength("//server/share") === 8      // "//server/"
 * getRootLength("\\\\server") === 7          // "\\\\server"
 * getRootLength("\\\\server\\share") === 8   // "\\\\server\\"
 * getRootLength("file:///path") === 8        // "file:///"
 * getRootLength("file:///c:") === 10         // "file:///c:"
 * getRootLength("file:///c:d") === 8         // "file:///"
 * getRootLength("file:///c:/path") === 11    // "file:///c:/"
 * getRootLength("file://server") === 13      // "file://server"
 * getRootLength("file://server/path") === 14 // "file://server/"
 * getRootLength("http://server") === 13      // "http://server"
 * getRootLength("http://server/path") === 14 // "http://server/"
 * ```
 *
 * @internal
 */
function getRootLength(path: string) {
    const rootLength = getEncodedRootLength(path);
    return rootLength < 0 ? ~rootLength : rootLength;
}

/**
 * Returns the path except for its containing directory name.
 * Semantics align with NodeJS's `path.basename` except that we support URL's as well.
 *
 * ```ts
 * // POSIX
 * getBaseFileName("/path/to/file.ext") === "file.ext"
 * getBaseFileName("/path/to/") === "to"
 * getBaseFileName("/") === ""
 * // DOS
 * getBaseFileName("c:/path/to/file.ext") === "file.ext"
 * getBaseFileName("c:/path/to/") === "to"
 * getBaseFileName("c:/") === ""
 * getBaseFileName("c:") === ""
 * // URL
 * getBaseFileName("http://typescriptlang.org/path/to/file.ext") === "file.ext"
 * getBaseFileName("http://typescriptlang.org/path/to/") === "to"
 * getBaseFileName("http://typescriptlang.org/") === ""
 * getBaseFileName("http://typescriptlang.org") === ""
 * getBaseFileName("file://server/path/to/file.ext") === "file.ext"
 * getBaseFileName("file://server/path/to/") === "to"
 * getBaseFileName("file://server/") === ""
 * getBaseFileName("file://server") === ""
 * getBaseFileName("file:///path/to/file.ext") === "file.ext"
 * getBaseFileName("file:///path/to/") === "to"
 * getBaseFileName("file:///") === ""
 * getBaseFileName("file://") === ""
 * ```
 *
 * @internal
 */
export function getBaseFileName(path: string): string;
/**
 * Gets the portion of a path following the last (non-terminal) separator (`/`).
 * Semantics align with NodeJS's `path.basename` except that we support URL's as well.
 * If the base name has any one of the provided extensions, it is removed.
 *
 * ```ts
 * getBaseFileName("/path/to/file.ext", ".ext", true) === "file"
 * getBaseFileName("/path/to/file.js", ".ext", true) === "file.js"
 * getBaseFileName("/path/to/file.js", [".ext", ".js"], true) === "file"
 * getBaseFileName("/path/to/file.ext", ".EXT", false) === "file.ext"
 * ```
 *
 * @internal
 */
export function getBaseFileName(path: string, extensions: string | readonly string[], ignoreCase: boolean): string;
/** @internal */
export function getBaseFileName(path: string, extensions?: string | readonly string[], ignoreCase?: boolean) {
    path = normalizeSlashes(path);

    // if the path provided is itself the root, then it has not file name.
    const rootLength = getRootLength(path);
    if (rootLength === path.length) return "";

    // return the trailing portion of the path starting after the last (non-terminal) directory
    // separator but not including any trailing directory separator.
    path = removeTrailingDirectorySeparator(path);
    const name = path.slice(Math.max(getRootLength(path), path.lastIndexOf(directorySeparator) + 1));
    const extension = extensions !== undefined && ignoreCase !== undefined ? getAnyExtensionFromPath(name, extensions, ignoreCase) : undefined;
    return extension ? name.slice(0, name.length - extension.length) : name;
}

function tryGetExtensionFromPath(path: string, extension: string, stringEqualityComparer: (a: string, b: string) => boolean) {
    if (!startsWith(extension, ".")) extension = "." + extension;
    if (path.length >= extension.length && path.charCodeAt(path.length - extension.length) === CharacterCodes.dot) {
        const pathExtension = path.slice(path.length - extension.length);
        if (stringEqualityComparer(pathExtension, extension)) {
            return pathExtension;
        }
    }
}

function getAnyExtensionFromPathWorker(path: string, extensions: string | readonly string[], stringEqualityComparer: (a: string, b: string) => boolean) {
    if (typeof extensions === "string") {
        return tryGetExtensionFromPath(path, extensions, stringEqualityComparer) || "";
    }
    for (const extension of extensions) {
        const result = tryGetExtensionFromPath(path, extension, stringEqualityComparer);
        if (result) return result;
    }
    return "";
}

/**
 * Gets the file extension for a path.
 *
 * ```ts
 * getAnyExtensionFromPath("/path/to/file.ext") === ".ext"
 * getAnyExtensionFromPath("/path/to/file.ext/") === ".ext"
 * getAnyExtensionFromPath("/path/to/file") === ""
 * getAnyExtensionFromPath("/path/to.ext/file") === ""
 * ```
 *
 * @internal
 */
export function getAnyExtensionFromPath(path: string): string;
/**
 * Gets the file extension for a path, provided it is one of the provided extensions.
 *
 * ```ts
 * getAnyExtensionFromPath("/path/to/file.ext", ".ext", true) === ".ext"
 * getAnyExtensionFromPath("/path/to/file.js", ".ext", true) === ""
 * getAnyExtensionFromPath("/path/to/file.js", [".ext", ".js"], true) === ".js"
 * getAnyExtensionFromPath("/path/to/file.ext", ".EXT", false) === ""
 *
 * @internal
 */
export function getAnyExtensionFromPath(path: string, extensions: string | readonly string[], ignoreCase: boolean): string;
/** @internal */
export function getAnyExtensionFromPath(path: string, extensions?: string | readonly string[], ignoreCase?: boolean): string {
    // Retrieves any string from the final "." onwards from a base file name.
    // Unlike extensionFromPath, which throws an exception on unrecognized extensions.
    if (extensions) {
        return getAnyExtensionFromPathWorker(removeTrailingDirectorySeparator(path), extensions, ignoreCase ? equateStringsCaseInsensitive : equateStringsCaseSensitive);
    }
    const baseFileName = getBaseFileName(path);
    const extensionIndex = baseFileName.lastIndexOf(".");
    if (extensionIndex >= 0) {
        return baseFileName.substring(extensionIndex);
    }
    return "";
}

function pathComponents(path: string, rootLength: number) {
    const root = path.substring(0, rootLength);
    const rest = path.substring(rootLength).split(directorySeparator);
    if (rest.length && !lastOrUndefined(rest)) rest.pop();
    return [root, ...rest];
}

/** @internal */
type PathPathComponents = Path[] & { __pathComponensBrand: any; };

/**
 * Parse a path into an array containing a root component (at index 0) and zero or more path
 * components (at indices > 0). The result is not normalized.
 * If the path is relative, the root component is `""`.
 * If the path is absolute, the root component includes the first path separator (`/`).
 *
 * ```ts
 * // POSIX
 * getPathComponents("/path/to/file.ext") === ["/", "path", "to", "file.ext"]
 * getPathComponents("/path/to/") === ["/", "path", "to"]
 * getPathComponents("/") === ["/"]
 * // DOS
 * getPathComponents("c:/path/to/file.ext") === ["c:/", "path", "to", "file.ext"]
 * getPathComponents("c:/path/to/") === ["c:/", "path", "to"]
 * getPathComponents("c:/") === ["c:/"]
 * getPathComponents("c:") === ["c:"]
 * // URL
 * getPathComponents("http://typescriptlang.org/path/to/file.ext") === ["http://typescriptlang.org/", "path", "to", "file.ext"]
 * getPathComponents("http://typescriptlang.org/path/to/") === ["http://typescriptlang.org/", "path", "to"]
 * getPathComponents("http://typescriptlang.org/") === ["http://typescriptlang.org/"]
 * getPathComponents("http://typescriptlang.org") === ["http://typescriptlang.org"]
 * getPathComponents("file://server/path/to/file.ext") === ["file://server/", "path", "to", "file.ext"]
 * getPathComponents("file://server/path/to/") === ["file://server/", "path", "to"]
 * getPathComponents("file://server/") === ["file://server/"]
 * getPathComponents("file://server") === ["file://server"]
 * getPathComponents("file:///path/to/file.ext") === ["file:///", "path", "to", "file.ext"]
 * getPathComponents("file:///path/to/") === ["file:///", "path", "to"]
 * getPathComponents("file:///") === ["file:///"]
 * getPathComponents("file://") === ["file://"]
 * ```
 *
 * @internal
 */
function getPathComponents(path: Path): PathPathComponents;
/** @internal */
function getPathComponents(path: string, currentDirectory?: string): string[];
function getPathComponents(path: string, currentDirectory = "") {
    path = combinePaths(currentDirectory, path);
    return pathComponents(path, getRootLength(path));
}

//// Path Formatting

/**
 * Formats a parsed path consisting of a root component (at index 0) and zero or more path
 * segments (at indices > 0).
 *
 * ```ts
 * getPathFromPathComponents(["/", "path", "to", "file.ext"]) === "/path/to/file.ext"
 * ```
 *
 * @internal
 */
function getPathFromPathComponents<T extends string>(pathComponents: readonly T[], length?: number) {
    if (pathComponents.length === 0) return "" as T;

    const root = pathComponents[0] && ensureTrailingDirectorySeparator(pathComponents[0]);
    return root + pathComponents.slice(1, length).join(directorySeparator) as T;
}

//// Path Normalization

/**
 * Normalize path separators, converting `\` into `/`.
 *
 * @internal
 */
function normalizeSlashes(path: string): string {
    return path.includes("\\")
        ? path.replace(backslashRegExp, directorySeparator)
        : path;
}

/**
 * Reduce an array of path components to a more simplified path by navigating any
 * `"."` or `".."` entries in the path.
 *
 * @internal
 */
function reducePathComponents(components: readonly string[]) {
    if (!some(components)) return [];
    const reduced = [components[0]];
    for (let i = 1; i < components.length; i++) {
        const component = components[i];
        if (!component) continue;
        if (component === ".") continue;
        if (component === "..") {
            if (reduced.length > 1) {
                if (reduced[reduced.length - 1] !== "..") {
                    reduced.pop();
                    continue;
                }
            }
            else if (reduced[0]) continue;
        }
        reduced.push(component);
    }
    return reduced;
}

/**
 * Combines paths. If a path is absolute, it replaces any previous path. Relative paths are not simplified.
 *
 * ```ts
 * // Non-rooted
 * combinePaths("path", "to", "file.ext") === "path/to/file.ext"
 * combinePaths("path", "dir", "..", "to", "file.ext") === "path/dir/../to/file.ext"
 * // POSIX
 * combinePaths("/path", "to", "file.ext") === "/path/to/file.ext"
 * combinePaths("/path", "/to", "file.ext") === "/to/file.ext"
 * // DOS
 * combinePaths("c:/path", "to", "file.ext") === "c:/path/to/file.ext"
 * combinePaths("c:/path", "c:/to", "file.ext") === "c:/to/file.ext"
 * // URL
 * combinePaths("file:///path", "to", "file.ext") === "file:///path/to/file.ext"
 * combinePaths("file:///path", "file:///to", "file.ext") === "file:///to/file.ext"
 * ```
 *
 * @internal
 */
function combinePaths(path: string, ...paths: (string | undefined)[]): string {
    if (path) path = normalizeSlashes(path);
    for (let relativePath of paths) {
        if (!relativePath) continue;
        relativePath = normalizeSlashes(relativePath);
        if (!path || getRootLength(relativePath) !== 0) {
            path = relativePath;
        }
        else {
            path = ensureTrailingDirectorySeparator(path) + relativePath;
        }
    }
    return path;
}

/** @internal */
export function normalizePath(path: string): string {
    path = normalizeSlashes(path);
    // Most paths don't require normalization
    if (!relativePathSegmentRegExp.test(path)) {
        return path;
    }
    // Some paths only require cleanup of `/./` or leading `./`
    const simplified = path.replace(/\/\.\//g, "/").replace(/^\.\//, "");
    if (simplified !== path) {
        path = simplified;
        if (!relativePathSegmentRegExp.test(path)) {
            return path;
        }
    }
    // Other paths require full normalization
    const normalized = getPathFromPathComponents(reducePathComponents(getPathComponents(path)));
    return normalized && hasTrailingDirectorySeparator(path) ? ensureTrailingDirectorySeparator(normalized) : normalized;
}

//// Path Mutation

/**
 * Removes a trailing directory separator from a path, if it does not already have one.
 *
 * ```ts
 * removeTrailingDirectorySeparator("/path/to/file.ext") === "/path/to/file.ext"
 * removeTrailingDirectorySeparator("/path/to/file.ext/") === "/path/to/file.ext"
 * ```
 *
 * @internal
 */
function removeTrailingDirectorySeparator(path: Path): Path;
/** @internal */
function removeTrailingDirectorySeparator(path: string): string;
/** @internal */
function removeTrailingDirectorySeparator(path: string) {
    if (hasTrailingDirectorySeparator(path)) {
        return path.substr(0, path.length - 1);
    }

    return path;
}

/**
 * Adds a trailing directory separator to a path, if it does not already have one.
 *
 * ```ts
 * ensureTrailingDirectorySeparator("/path/to/file.ext") === "/path/to/file.ext/"
 * ensureTrailingDirectorySeparator("/path/to/file.ext/") === "/path/to/file.ext/"
 * ```
 *
 * @internal
 */
function ensureTrailingDirectorySeparator(path: Path): Path;
/** @internal */
function ensureTrailingDirectorySeparator(path: string): string;
/** @internal */
function ensureTrailingDirectorySeparator(path: string) {
    if (!hasTrailingDirectorySeparator(path)) {
        return path + directorySeparator;
    }

    return path;
}

//// Path Comparisons

// check path for these segments: '', '.'. '..'
const relativePathSegmentRegExp = /(?:\/\/)|(?:^|\/)\.\.?(?:$|\/)/;

//// Relative Paths

//// Path Traversal
