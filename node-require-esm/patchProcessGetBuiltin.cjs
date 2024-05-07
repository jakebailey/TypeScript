const _module = require("module");

/** @type {(name: string) => any} */
function getBuiltinModule(name) {
    if (!_module.isBuiltin(name)) return undefined;
    return require(name);
}

process.getBuiltinModule = getBuiltinModule;
