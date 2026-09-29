// @target: es2015, esnext
// @module: commonjs, esnext
// @strict: true
// @sourceMap: true

// @filename: constants.ts
export const enum Values {
    First = 1,
    Last = First + 2,
    Name = "value"
}
export { Values as Alias };

// @filename: main.ts
import { Alias, Values } from "./constants";
export const first = Values.First;
export const record = { [Alias.Name]: Alias["Last"] };
export async function read() {
    return [Values.First, Alias.Last, Values.Name];
}
export const access = record?.[Values.Name] ?? Values.Last;

// @filename: script.ts
const enum ScriptValues { Value = 42 }
const scriptResult = ScriptValues.Value;
