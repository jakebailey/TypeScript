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
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
//# sourceMappingURL=constants.js.map
//// [main.js]
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.access = exports.record = exports.first = void 0;
exports.read = read;
exports.first = 1 /* Values.First */;
exports.record = { ["value" /* Alias.Name */]: 3 /* Alias["Last"] */ };
async function read() {
    return [1 /* Values.First */, 3 /* Alias.Last */, "value" /* Values.Name */];
}
exports.access = exports.record?.["value" /* Values.Name */] ?? 3 /* Values.Last */;
//# sourceMappingURL=main.js.map
//// [script.js]
"use strict";
const scriptResult = 42 /* ScriptValues.Value */;
//# sourceMappingURL=script.js.map