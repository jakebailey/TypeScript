/**
 * Worker-side bridge between the synchronous API client and the API subprocess.
 */
import {
    type ChildProcess,
    spawn,
} from "node:child_process";
import { workerData } from "node:worker_threads";

interface SyncWorkerData {
    exe: string;
    args: string[];
    controlBuffer: SharedArrayBuffer;
    dataBuffer: SharedArrayBuffer;
}

interface Message {
    type: number;
    name: Buffer;
    payload: Buffer;
}

const STATE_INITIALIZING = -1;
const STATE_IDLE = 0;
const STATE_REQUEST_PENDING = 1;
const STATE_RESPONSE_READY = 2;
const STATE_ERROR = 3;
const STATE_SHUTDOWN = 4;
const STATE_CALLBACK = 5;
const STATE_CLOSED = 6;
const STATE_RESPONSE_ACKNOWLEDGED = 7;

const CONTROL_STATE = 0;
const CONTROL_REQUEST_TYPE = 1;
const CONTROL_REQUEST_NAME_LENGTH = 2;
const CONTROL_REQUEST_PAYLOAD_LENGTH = 3;
const CONTROL_RESPONSE_TYPE = 4;
const CONTROL_RESPONSE_NAME_LENGTH = 5;
const CONTROL_RESPONSE_PAYLOAD_LENGTH = 6;

const MSG_CALL_RESPONSE = 2;
const MSG_CALL_ERROR = 3;
const MSG_RESPONSE = 4;
const MSG_ERROR = 5;
const MSG_CALL = 6;

const MSGPACK_FIXARRAY3 = 0x93;
const MSGPACK_BIN8 = 0xc4;
const MSGPACK_BIN16 = 0xc5;
const MSGPACK_BIN32 = 0xc6;
const MSGPACK_UINT8 = 0xcc;

const {
    exe,
    args,
    controlBuffer,
    dataBuffer,
} = workerData as SyncWorkerData;
const control = new Int32Array(controlBuffer);
const data = new Uint8Array(dataBuffer);

let child: ChildProcess | undefined;
let readBuffer: Buffer<ArrayBufferLike> = Buffer.alloc(0);
let messageResolve: ((message: Message) => void) | undefined;
let messageReject: ((error: Error) => void) | undefined;

function encodedBinSize(length: number): number {
    if (length < 0x100) return 2 + length;
    if (length < 0x10000) return 3 + length;
    return 5 + length;
}

function writeBin(buffer: Buffer, offset: number, value: Uint8Array): number {
    if (value.length < 0x100) {
        buffer[offset++] = MSGPACK_BIN8;
        buffer[offset++] = value.length;
    }
    else if (value.length < 0x10000) {
        buffer[offset++] = MSGPACK_BIN16;
        buffer.writeUInt16BE(value.length, offset);
        offset += 2;
    }
    else {
        buffer[offset++] = MSGPACK_BIN32;
        buffer.writeUInt32BE(value.length, offset);
        offset += 4;
    }
    buffer.set(value, offset);
    return offset + value.length;
}

function encodeMessage(type: number, name: Uint8Array, payload: Uint8Array): Buffer {
    const buffer = Buffer.allocUnsafe(2 + encodedBinSize(name.length) + encodedBinSize(payload.length));
    let offset = 0;
    buffer[offset++] = MSGPACK_FIXARRAY3;
    buffer[offset++] = type;
    offset = writeBin(buffer, offset, name);
    writeBin(buffer, offset, payload);
    return buffer;
}

function readBin(buffer: Buffer, offset: number): { value: Buffer; offset: number; } | undefined {
    if (buffer.length <= offset) return undefined;

    const marker = buffer[offset++];
    let length: number;
    switch (marker) {
        case MSGPACK_BIN8:
            if (buffer.length < offset + 1) return undefined;
            length = buffer[offset++];
            break;
        case MSGPACK_BIN16:
            if (buffer.length < offset + 2) return undefined;
            length = buffer.readUInt16BE(offset);
            offset += 2;
            break;
        case MSGPACK_BIN32:
            if (buffer.length < offset + 4) return undefined;
            length = buffer.readUInt32BE(offset);
            offset += 4;
            break;
        default:
            throw new Error(`Expected binary data (0xc4-0xc6), received: 0x${marker.toString(16)}`);
    }

    if (buffer.length < offset + length) return undefined;
    return {
        value: buffer.subarray(offset, offset + length),
        offset: offset + length,
    };
}

function tryReadMessage(): Message | undefined {
    if (readBuffer.length < 2) return undefined;
    if (readBuffer[0] !== MSGPACK_FIXARRAY3) {
        throw new Error(`Expected fixed 3-element array (0x93), received: 0x${readBuffer[0].toString(16)}`);
    }

    let offset = 1;
    let type = readBuffer[offset++];
    if (type === MSGPACK_UINT8) {
        if (readBuffer.length <= offset) return undefined;
        type = readBuffer[offset++];
    }
    else if (type > 0x7f) {
        throw new Error(`Expected positive fixint or uint8 marker, received: 0x${type.toString(16)}`);
    }

    const name = readBin(readBuffer, offset);
    if (name === undefined) return undefined;
    const payload = readBin(readBuffer, name.offset);
    if (payload === undefined) return undefined;

    readBuffer = readBuffer.subarray(payload.offset);
    return {
        type,
        name: Buffer.from(name.value),
        payload: Buffer.from(payload.value),
    };
}

function processBuffer(): void {
    if (messageResolve === undefined) return;
    try {
        const message = tryReadMessage();
        if (message === undefined) return;
        const resolve = messageResolve;
        messageResolve = undefined;
        messageReject = undefined;
        resolve(message);
    }
    catch (error) {
        rejectMessage(error);
    }
}

function rejectMessage(error: unknown): void {
    if (messageReject === undefined) return;
    const reject = messageReject;
    messageResolve = undefined;
    messageReject = undefined;
    reject(error instanceof Error ? error : new Error(String(error)));
}

function waitForMessage(): Promise<Message> {
    if (messageResolve !== undefined) {
        throw new Error("Already waiting for an API subprocess message");
    }
    return new Promise((resolve, reject) => {
        messageResolve = resolve;
        messageReject = reject;
        processBuffer();
    });
}

function sendToChild(type: number, name: Uint8Array, payload: Uint8Array): void {
    if (child?.stdin === null || child?.stdin === undefined || child.stdin.destroyed) {
        throw new Error("API subprocess is not available");
    }
    child.stdin.write(encodeMessage(type, name, payload));
}

function checkDataSize(nameLength: number, payloadLength: number): void {
    if (nameLength < 0 || payloadLength < 0 || nameLength + payloadLength > data.length) {
        throw new Error(`Synchronous API message is too large (${nameLength + payloadLength} bytes; maximum ${data.length})`);
    }
}

function readRequest(): Message {
    const nameLength = control[CONTROL_REQUEST_NAME_LENGTH];
    const payloadLength = control[CONTROL_REQUEST_PAYLOAD_LENGTH];
    checkDataSize(nameLength, payloadLength);
    return {
        type: control[CONTROL_REQUEST_TYPE],
        name: Buffer.from(data.subarray(0, nameLength)),
        payload: Buffer.from(data.subarray(nameLength, nameLength + payloadLength)),
    };
}

function writeResponse(type: number, name: Uint8Array, payload: Uint8Array): void {
    checkDataSize(name.length, payload.length);
    control[CONTROL_RESPONSE_TYPE] = type;
    control[CONTROL_RESPONSE_NAME_LENGTH] = name.length;
    control[CONTROL_RESPONSE_PAYLOAD_LENGTH] = payload.length;
    data.set(name, 0);
    data.set(payload, name.length);
}

function writeError(error: unknown, name: Uint8Array = Buffer.alloc(0)): void {
    const message = Buffer.from(error instanceof Error ? error.message : String(error));
    const available = data.length - name.length;
    writeResponse(MSG_ERROR, name, message.subarray(0, Math.max(0, available)));
}

async function processResponses(expectedName: Buffer): Promise<void> {
    for (;;) {
        const message = await waitForMessage();
        if (message.type === MSG_RESPONSE || message.type === MSG_ERROR) {
            if (!message.name.equals(expectedName)) {
                throw new Error(`name mismatch for response: expected \`${expectedName.toString("utf8")}\`, got \`${message.name.toString("utf8")}\``);
            }
            writeResponse(message.type, message.name, message.payload);
            return;
        }
        if (message.type !== MSG_CALL) {
            throw new Error(`Invalid message type from child: ${message.type}`);
        }

        writeResponse(message.type, message.name, message.payload);
        Atomics.store(control, CONTROL_STATE, STATE_CALLBACK);
        Atomics.notify(control, CONTROL_STATE);
        Atomics.wait(control, CONTROL_STATE, STATE_CALLBACK);

        const state = Atomics.load(control, CONTROL_STATE);
        if (state === STATE_SHUTDOWN) {
            throw new Error("Synchronous API worker shut down during a callback");
        }
        const callbackResponse = readRequest();
        if (callbackResponse.type !== MSG_CALL_RESPONSE && callbackResponse.type !== MSG_CALL_ERROR) {
            throw new Error(`Invalid callback response type: ${callbackResponse.type}`);
        }
        sendToChild(callbackResponse.type, callbackResponse.name, callbackResponse.payload);
    }
}

async function handleRequest(): Promise<void> {
    const request = readRequest();
    try {
        sendToChild(request.type, request.name, request.payload);
        await processResponses(request.name);
        Atomics.store(control, CONTROL_STATE, STATE_RESPONSE_READY);
    }
    catch (error) {
        writeError(error, request.name);
        Atomics.store(control, CONTROL_STATE, STATE_ERROR);
    }
    Atomics.notify(control, CONTROL_STATE);
}

async function spawnChild(): Promise<void> {
    child = spawn(exe, args, {
        stdio: ["pipe", "pipe", "inherit"],
    });

    child.stdout!.on("data", (chunk: Buffer) => {
        readBuffer = readBuffer.length === 0 ? chunk : Buffer.concat([readBuffer, chunk]);
        processBuffer();
    });
    child.stdout!.on("error", rejectMessage);
    child.on("error", rejectMessage);
    child.on("exit", (code, signal) => {
        rejectMessage(new Error(signal === null ? `API subprocess exited with code ${code}` : `API subprocess was killed by signal ${signal}`));
    });

    await new Promise<void>((resolve, reject) => {
        child!.once("spawn", resolve);
        child!.once("error", reject);
    });
}

async function main(): Promise<void> {
    await spawnChild();
    Atomics.store(control, CONTROL_STATE, STATE_IDLE);
    Atomics.notify(control, CONTROL_STATE);

    for (;;) {
        Atomics.wait(control, CONTROL_STATE, STATE_IDLE);
        let state = Atomics.load(control, CONTROL_STATE);
        if (state === STATE_SHUTDOWN) break;
        if (state !== STATE_REQUEST_PENDING) {
            throw new Error(`Unexpected synchronous API channel state: ${state}`);
        }

        await handleRequest();
        state = Atomics.load(control, CONTROL_STATE);
        if (state === STATE_RESPONSE_READY || state === STATE_ERROR) {
            Atomics.wait(control, CONTROL_STATE, state);
        }
        state = Atomics.load(control, CONTROL_STATE);
        if (state === STATE_SHUTDOWN) break;
        if (state !== STATE_RESPONSE_ACKNOWLEDGED) {
            throw new Error(`Unexpected synchronous API channel state after response: ${state}`);
        }
        Atomics.store(control, CONTROL_STATE, STATE_IDLE);
        Atomics.notify(control, CONTROL_STATE);
    }

    child?.stdin?.end();
    child?.kill();
    Atomics.store(control, CONTROL_STATE, STATE_CLOSED);
    Atomics.notify(control, CONTROL_STATE);
}

void main().catch(error => {
    child?.kill();
    writeError(error);
    Atomics.store(control, CONTROL_STATE, STATE_ERROR);
    Atomics.notify(control, CONTROL_STATE);
});
