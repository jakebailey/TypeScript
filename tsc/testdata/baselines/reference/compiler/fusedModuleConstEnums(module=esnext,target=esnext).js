//// [tests/cases/compiler/fusedModuleConstEnums.ts] ////

//// [constants.ts]
export const enum Values {
    First = 1,
    Last = First + 2,
    Name = "value"
}
export { Values as Alias };

//// [main.ts]
import { Alias, Values } from "./constants";
export const first = Values.First;
export const record = { [Alias.Name]: Alias["Last"] };
export async function read() {
    return [Values.First, Alias.Last, Values.Name];
}
export const access = record?.[Values.Name] ?? Values.Last;

//// [script.ts]
const enum ScriptValues { Value = 42 }
const scriptResult = ScriptValues.Value;


//// [constants.js]
export {};
//# sourceMappingURL=constants.js.map
//// [main.js]
export const first = 1 /* Values.First */;
export const record = { ["value" /* Alias.Name */]: 3 /* Alias["Last"] */ };
export async function read() {
    return [1 /* Values.First */, 3 /* Alias.Last */, "value" /* Values.Name */];
}
export const access = record?.["value" /* Values.Name */] ?? 3 /* Values.Last */;
//# sourceMappingURL=main.js.map
//// [script.js]
"use strict";
const scriptResult = 42 /* ScriptValues.Value */;
//# sourceMappingURL=script.js.map