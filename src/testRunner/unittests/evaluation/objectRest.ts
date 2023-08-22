import * as evaluator from "../../_namespaces/evaluator";

describe("unittests:: evaluation:: objectRest", () => {
    // https://github.com/microsoft/TypeScript/issues/31469
    it("side effects in property assignment", async () => {
        const result = evaluator.evaluateTypeScript(`
            const k: any = { a: 1, b: 2 };
            const o: any = { a: 3, ...k, b: k.a++ };
            export const output = o;
        `);
        assert.deepEqual(result.output, { a: 1, b: 1 });
    });
    it("side effects in during spread", async () => {
        const result = evaluator.evaluateTypeScript(`
            const k: any = { a: 1, get b() { l = { c: 9 }; return 2; } };
            let l: any = { c: 3 };
            const o: any = { ...k, ...l };
            export const output = o;
        `);
        assert.deepEqual(result.output, { a: 1, b: 2, c: 9 });
    });
    it("trailing literal-valued object-literal", async () => {
        const result = evaluator.evaluateTypeScript(`
            const k: any = { a: 1 }
            const o: any = { ...k, ...{ b: 2 } };
            export const output = o;
        `);
        assert.deepEqual(result.output, { a: 1, b: 2 });
    });
});
