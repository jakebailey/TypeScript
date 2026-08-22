import assert from "node:assert";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { Worker } from "node:worker_threads";
import { SyncRpcChannel } from "../../src/api/syncChannel.ts";

const repoRoot = fileURLToPath(new URL("../../../../", import.meta.url));
const tscPath = fileURLToPath(new URL(`../../../../built/local/tsc${process.platform === "win32" ? ".exe" : ""}`, import.meta.url));

test("worker transport handles large binary messages", () => {
    const channel = new SyncRpcChannel(tscPath, ["--api", "--cwd", repoRoot]);
    try {
        const input = new Uint8Array(1024 * 1024);
        for (let i = 0; i < input.length; i++) {
            input[i] = i & 0xff;
        }
        assert.deepEqual(channel.requestBinarySync("echo", input), input);
    }
    finally {
        channel.close();
    }
});

test("worker transport supports concurrent API clients", async () => {
    const apiURL = new URL("../../src/api/sync/api.ts", import.meta.url).href;
    const fsURL = new URL("../../src/api/fs.ts", import.meta.url).href;
    const results = await Promise.all(
        Array.from({ length: 4 }, (_, index) => runAPIWorker(index, apiURL, fsURL)),
    );
    assert.deepEqual(results, [
        "export const value = 0;",
        "export const value = 1;",
        "export const value = 2;",
        "export const value = 3;",
    ]);
});

function runAPIWorker(index: number, apiURL: string, fsURL: string): Promise<string> {
    const sourceText = `export const value = ${index};`;
    const script = `
        const { parentPort, workerData } = require("node:worker_threads");
        void (async () => {
            const [{ API }, { createVirtualFileSystem }] = await Promise.all([
                import(workerData.apiURL),
                import(workerData.fsURL),
            ]);
            const api = new API({
                cwd: workerData.cwd,
                tsserverPath: workerData.tscPath,
                fs: createVirtualFileSystem({
                    "/tsconfig.json": "{}",
                    "/index.ts": workerData.sourceText,
                }),
            });
            try {
                const snapshot = api.updateSnapshot({ openProject: "/tsconfig.json" });
                const project = snapshot.getProject("/tsconfig.json");
                const sourceFile = project.program.getSourceFile("/index.ts");
                parentPort.postMessage(sourceFile.text);
            }
            finally {
                api.close();
            }
        })().catch(error => {
            throw error;
        });
    `;

    return new Promise((resolve, reject) => {
        const worker = new Worker(script, {
            eval: true,
            workerData: {
                apiURL,
                cwd: repoRoot,
                fsURL,
                sourceText,
                tscPath,
            },
        });
        let result: string | undefined;
        worker.once("message", value => {
            result = value;
        });
        worker.once("error", reject);
        worker.once("exit", code => {
            if (code !== 0) {
                reject(new Error(`API worker exited with code ${code}`));
            }
            else if (result === undefined) {
                reject(new Error("API worker exited without a result"));
            }
            else {
                resolve(result);
            }
        });
    });
}
