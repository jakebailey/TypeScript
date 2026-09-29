// @target: es2015, es2022
// @module: commonjs, esnext
// @lib: esnext
// @strict: true
// @sourceMap: true

export function read() {
    return [counter, M.value, C.value];
}
export { counter as alias };
export let counter = 0;

export namespace M {
    export interface TypeOnly {}
}
export namespace M {
    export const value = 1;
}

function decorate<T extends new (...args: any[]) => any>(value: T) {
    return value;
}

using resource = { [Symbol.dispose]() { counter += 1; } };

@decorate
export class C {
    static value = counter ||= 2;
    #value = M.value;
    async run(input?: { value?: number }) {
        const { value, ...rest } = input ?? {};
        await Promise.resolve(rest);
        return value ?? this.#value;
    }
}

@decorate
export default class {
    value = C.value;
}

export const result = read();
