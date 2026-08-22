/**
 * Synchronous RPC channel backed by a worker thread.
 *
 * The main thread exchanges requests with the worker through shared memory so
 * it can block synchronously. The worker owns the subprocess and performs its
 * pipe I/O asynchronously.
 */
import { Worker } from "node:worker_threads";

const STATE_INITIALIZING = -1;
const STATE_IDLE = 0;
const STATE_REQUEST_PENDING = 1;
const STATE_RESPONSE_READY = 2;
const STATE_ERROR = 3;
const STATE_SHUTDOWN = 4;
const STATE_CALLBACK = 5;
const STATE_CLOSED = 6;

const CONTROL_STATE = 0;
const CONTROL_REQUEST_TYPE = 1;
const CONTROL_REQUEST_NAME_LENGTH = 2;
const CONTROL_REQUEST_PAYLOAD_LENGTH = 3;
const CONTROL_RESPONSE_TYPE = 4;
const CONTROL_RESPONSE_NAME_LENGTH = 5;
const CONTROL_RESPONSE_PAYLOAD_LENGTH = 6;
const CONTROL_BUFFER_SIZE = 7 * Int32Array.BYTES_PER_ELEMENT;
const DATA_BUFFER_SIZE = 64 * 1024 * 1024;

const MSG_REQUEST = 1;
const MSG_CALL_RESPONSE = 2;
const MSG_CALL_ERROR = 3;
const MSG_RESPONSE = 4;
const MSG_ERROR = 5;

export type CallbackFn = (name: string, payload: string) => string;

export class SyncRpcChannel {
    private readonly worker: Worker;
    private readonly control: Int32Array;
    private readonly data: Uint8Array;
    private readonly callbacks = new Map<string, CallbackFn>();
    private readonly encoder = new TextEncoder();
    private readonly decoder = new TextDecoder();
    private readonly collectTiming: boolean;
    private closed = false;

    lastBytesSent = 0;
    lastBytesReceived = 0;

    constructor(exe: string, args: string[], collectTiming = false) {
        this.collectTiming = collectTiming;

        const controlBuffer = new SharedArrayBuffer(CONTROL_BUFFER_SIZE);
        const dataBuffer = new SharedArrayBuffer(DATA_BUFFER_SIZE);
        this.control = new Int32Array(controlBuffer);
        this.data = new Uint8Array(dataBuffer);
        Atomics.store(this.control, CONTROL_STATE, STATE_INITIALIZING);

        const workerPath = new URL(import.meta.url.endsWith(".ts") ? "./syncWorker.ts" : "./syncWorker.js", import.meta.url);
        this.worker = new Worker(workerPath, {
            workerData: {
                exe,
                args,
                controlBuffer,
                dataBuffer,
            },
        });
        this.worker.unref();

        const waitResult = Atomics.wait(this.control, CONTROL_STATE, STATE_INITIALIZING, 30_000);
        if (waitResult === "timed-out") {
            void this.worker.terminate();
            throw new Error("Timed out waiting for synchronous API worker to initialize");
        }

        const state = Atomics.load(this.control, CONTROL_STATE);
        if (state !== STATE_IDLE) {
            const message = this.readResponsePayload();
            void this.worker.terminate();
            throw new Error(message.length === 0 ? `Synchronous API worker initialization failed with state ${state}` : this.decoder.decode(message));
        }
    }

    requestSync(method: string, payload: string): string {
        return this.decoder.decode(this.requestBytesSync(method, this.encoder.encode(payload)));
    }

    requestBinarySync(method: string, payload: Uint8Array): Uint8Array {
        return this.requestBytesSync(method, payload);
    }

    registerCallback(name: string, callback: CallbackFn): void {
        this.callbacks.set(name, callback);
    }

    close(): void {
        if (this.closed) return;
        this.closed = true;

        Atomics.store(this.control, CONTROL_STATE, STATE_SHUTDOWN);
        Atomics.notify(this.control, CONTROL_STATE);
        Atomics.wait(this.control, CONTROL_STATE, STATE_SHUTDOWN, 5_000);
        void this.worker.terminate();
    }

    private requestBytesSync(method: string, payload: Uint8Array): Uint8Array {
        if (this.closed) {
            throw new Error("SyncRpcChannel is closed");
        }
        const state = Atomics.load(this.control, CONTROL_STATE);
        if (state !== STATE_IDLE) {
            throw new Error(`SyncRpcChannel is not idle (state ${state})`);
        }

        const methodBytes = this.encoder.encode(method);
        if (this.collectTiming) {
            this.lastBytesSent = payload.length;
            this.lastBytesReceived = 0;
        }
        this.writeRequest(MSG_REQUEST, methodBytes, payload);
        Atomics.store(this.control, CONTROL_STATE, STATE_REQUEST_PENDING);
        Atomics.notify(this.control, CONTROL_STATE);

        for (;;) {
            let responseState = Atomics.load(this.control, CONTROL_STATE);
            if (responseState === STATE_REQUEST_PENDING) {
                Atomics.wait(this.control, CONTROL_STATE, STATE_REQUEST_PENDING);
                responseState = Atomics.load(this.control, CONTROL_STATE);
            }

            if (responseState === STATE_CALLBACK) {
                this.handleCallback();
                continue;
            }
            if (responseState === STATE_RESPONSE_READY || responseState === STATE_ERROR) {
                const responseType = this.control[CONTROL_RESPONSE_TYPE];
                const responseName = this.readResponseName();
                const responsePayload = this.readResponsePayload();
                Atomics.store(this.control, CONTROL_STATE, STATE_IDLE);
                Atomics.notify(this.control, CONTROL_STATE);

                if (responseState === STATE_ERROR || responseType === MSG_ERROR) {
                    throw new Error(this.decoder.decode(responsePayload));
                }
                if (responseType !== MSG_RESPONSE) {
                    throw new Error(`Invalid message type from child: ${responseType}`);
                }
                if (responseName !== method) {
                    throw new Error(`name mismatch for response: expected \`${method}\`, got \`${responseName}\``);
                }
                if (this.collectTiming) {
                    this.lastBytesReceived = responsePayload.length;
                }
                return responsePayload;
            }
            if (responseState === STATE_CLOSED) {
                throw new Error("Synchronous API worker closed before returning a response");
            }
            throw new Error(`Unexpected synchronous API worker state: ${responseState}`);
        }
    }

    private handleCallback(): void {
        const name = this.readResponseName();
        const payload = this.decoder.decode(this.readResponsePayload());
        const callback = this.callbacks.get(name);

        let responseType = MSG_CALL_RESPONSE;
        let response: string;
        if (callback === undefined) {
            responseType = MSG_CALL_ERROR;
            response = `unknown callback: \`${name}\`. Please make sure to register it on the JavaScript side before invoking it.`;
        }
        else {
            try {
                response = callback(name, payload);
            }
            catch (error) {
                responseType = MSG_CALL_ERROR;
                response = String(error instanceof Error ? error.message : error).trim();
            }
        }

        this.writeRequest(responseType, this.encoder.encode(name), this.encoder.encode(response));
        Atomics.store(this.control, CONTROL_STATE, STATE_REQUEST_PENDING);
        Atomics.notify(this.control, CONTROL_STATE);
    }

    private writeRequest(type: number, name: Uint8Array, payload: Uint8Array): void {
        this.checkDataSize(name.length, payload.length);
        this.control[CONTROL_REQUEST_TYPE] = type;
        this.control[CONTROL_REQUEST_NAME_LENGTH] = name.length;
        this.control[CONTROL_REQUEST_PAYLOAD_LENGTH] = payload.length;
        this.data.set(name, 0);
        this.data.set(payload, name.length);
    }

    private readResponseName(): string {
        const nameLength = this.control[CONTROL_RESPONSE_NAME_LENGTH];
        const payloadLength = this.control[CONTROL_RESPONSE_PAYLOAD_LENGTH];
        this.checkDataSize(nameLength, payloadLength);
        return this.decoder.decode(this.data.subarray(0, nameLength));
    }

    private readResponsePayload(): Uint8Array {
        const nameLength = this.control[CONTROL_RESPONSE_NAME_LENGTH];
        const payloadLength = this.control[CONTROL_RESPONSE_PAYLOAD_LENGTH];
        this.checkDataSize(nameLength, payloadLength);
        return this.data.slice(nameLength, nameLength + payloadLength);
    }

    private checkDataSize(nameLength: number, payloadLength: number): void {
        if (nameLength < 0 || payloadLength < 0 || nameLength + payloadLength > this.data.length) {
            throw new Error(`Synchronous API message is too large (${nameLength + payloadLength} bytes; maximum ${this.data.length})`);
        }
    }
}
