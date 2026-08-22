/**
 * NAPI-based client for the TypeScript Go API.
 *
 * This client loads the compiler directly as a native Node.js addon instead
 * of spawning a child process.
 */

import { createRequire } from "node:module";
import path from "node:path";
import { fileURLToPath } from "node:url";
import type {
    APIMethodInfo,
    SourceFileResponseMethod,
} from "../proto.ts";
import {
    combineTimingInfo,
    disabledTimingInfo,
    type ServerTimingInfo,
    TimingCollector,
    type TimingInfo,
} from "../timing.ts";

interface NapiModule {
    createSession(cwd: string, defaultLibraryPath?: string): void;
    request(method: string, payload: string): string;
    requestBinary(method: string, payload: Uint8Array): Uint8Array;
    close(): void;
}

function resolveNapiModulePath(explicitPath?: string): string {
    if (explicitPath) return explicitPath;

    const dirname = path.dirname(fileURLToPath(import.meta.url));
    const normalizedDirname = dirname.replace(/\\/g, "/");
    if (
        normalizedDirname.endsWith("/packages/typescript/src/api/sync")
        || normalizedDirname.endsWith("/packages/typescript/dist/api/sync")
    ) {
        const repoRoot = path.resolve(dirname, "..", "..", "..", "..", "..");
        return path.join(repoRoot, "built", "local", "tsgo.node");
    }
    return path.join(dirname, "..", "tsgo.node");
}

export interface NapiClientOptions {
    /** Current working directory. */
    cwd?: string;
    /** Explicit path to the tsgo.node native addon. */
    napiModulePath?: string;
    /**
     * Path to the directory containing bundled lib.d.ts files.
     * Required for noembed builds.
     */
    defaultLibraryPath?: string;
    /** Collect request and server timing information. */
    collectTiming?: boolean;
}

export class NapiClient {
    private module: NapiModule;
    private encoder = new TextEncoder();
    private timing: TimingCollector | undefined;

    constructor(options: NapiClientOptions = {}) {
        const cwd = options.cwd ?? process.cwd();
        const modulePath = path.resolve(resolveNapiModulePath(options.napiModulePath));
        const defaultLibraryPath = options.defaultLibraryPath ?? path.dirname(modulePath);
        const require = createRequire(import.meta.url);
        this.module = require(modulePath) as NapiModule;
        if (options.collectTiming) {
            this.timing = new TimingCollector();
        }
        this.module.createSession(cwd, defaultLibraryPath);
    }

    apiRequest<K extends keyof APIMethodInfo>(method: K, params?: APIMethodInfo[K]["params"]): APIMethodInfo[K]["result"] {
        const encodedPayload = JSON.stringify(params);
        const start = performance.now();
        const result = this.module.request(method, encodedPayload);
        this.recordTiming(method, start, encodedPayload.length, result.length);
        if (result.length) {
            return JSON.parse(result) as APIMethodInfo[K]["result"];
        }
        return undefined as APIMethodInfo[K]["result"];
    }

    apiRequestBinary<K extends SourceFileResponseMethod>(method: K, params?: APIMethodInfo[K]["params"]): Uint8Array | undefined {
        const payload = this.encoder.encode(JSON.stringify(params));
        const start = performance.now();
        const result = this.module.requestBinary(method, payload);
        this.recordTiming(method, start, payload.byteLength, result.byteLength);
        if (result.length === 0) return undefined;
        return result;
    }

    getTimingInfo(): TimingInfo {
        if (!this.timing) {
            return disabledTimingInfo();
        }
        const local = this.timing.getInfo();
        const result = this.module.request("getServerTiming", "");
        return combineTimingInfo(local, JSON.parse(result) as ServerTimingInfo);
    }

    resetTimingInfo(): void {
        if (!this.timing) return;
        this.timing.reset();
        this.module.request("resetServerTiming", "");
    }

    getTimingCollector(): TimingCollector | undefined {
        return this.timing;
    }

    private recordTiming(method: string, start: number, bytesSent: number, bytesReceived: number): void {
        this.timing?.record({
            method,
            roundTripMs: performance.now() - start,
            bytesSent,
            bytesReceived,
        });
    }

    echo(payload: string): string {
        return this.module.request("echo", payload);
    }

    echoBinary(payload: Uint8Array): Uint8Array {
        return this.module.requestBinary("echo", payload);
    }

    close(): void {
        this.module.close();
    }
}
