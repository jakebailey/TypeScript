/**
 * NAPI-backed TypeScript API.
 *
 * This module provides the same API surface as `@typescript/typescript/unstable/sync`,
 * but loads the compiler as a native Node.js addon instead of spawning a process.
 */

export * from "./api.ts";
export { NapiClient, type NapiClientOptions } from "./napiClient.ts";

import type {
    APIOptions,
    LSPConnectionOptions,
} from "../options.ts";
import {
    API as SyncAPI,
    type APIClient,
} from "./api.ts";
import {
    NapiClient,
    type NapiClientOptions,
} from "./napiClient.ts";

export class API extends SyncAPI {
    constructor(options: NapiClientOptions = {}) {
        super(options);
    }

    protected override createClient(options: APIOptions | LSPConnectionOptions): APIClient {
        if ("pipe" in options) {
            throw new Error("Socket connections are not supported by the NAPI client");
        }
        return new NapiClient(options);
    }
}
