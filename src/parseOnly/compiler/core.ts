import {
    Comparer,
    Comparison,
    EqualityComparer,
    MapLike,
    SortedArray,
    SortedReadonlyArray,
} from "./corePublic";
import {
    Debug,
} from "./debug";

/** @internal */
export const emptyArray: never[] = [] as never[];
/** @internal */
export const emptyMap: ReadonlyMap<never, never> = new Map<never, never>();

/** @internal */
export function length(array: readonly any[] | undefined): number {
    return array ? array.length : 0;
}

/**
 * Iterates through 'array' by index and performs the callback on each element of array until the callback
 * returns a truthy value, then returns that value.
 * If no such value is found, the callback is applied to each element of array and undefined is returned.
 *
 * @internal
 */
export function forEach<T, U>(array: readonly T[] | undefined, callback: (element: T, index: number) => U | undefined): U | undefined {
    if (array) {
        for (let i = 0; i < array.length; i++) {
            const result = callback(array[i], i);
            if (result) {
                return result;
            }
        }
    }
    return undefined;
}

/**
 * Iterates through `array` by index and performs the callback on each element of array until the callback
 * returns a falsey value, then returns false.
 * If no such value is found, the callback is applied to each element of array and `true` is returned.
 *
 * @internal
 */
export function every<T, U extends T>(array: readonly T[], callback: (element: T, index: number) => element is U): array is readonly U[];
/** @internal */
export function every<T, U extends T>(array: readonly T[] | undefined, callback: (element: T, index: number) => element is U): array is readonly U[] | undefined;
/** @internal */
export function every<T>(array: readonly T[] | undefined, callback: (element: T, index: number) => boolean): boolean;
export function every<T>(array: readonly T[] | undefined, callback: (element: T, index: number) => boolean): boolean {
    if (array) {
        for (let i = 0; i < array.length; i++) {
            if (!callback(array[i], i)) {
                return false;
            }
        }
    }

    return true;
}

/**
 * Works like Array.prototype.find, returning `undefined` if no element satisfying the predicate is found.
 *
 * @internal
 */
export function find<T, U extends T>(array: readonly T[] | undefined, predicate: (element: T, index: number) => element is U, startIndex?: number): U | undefined;
/** @internal */
export function find<T>(array: readonly T[] | undefined, predicate: (element: T, index: number) => boolean, startIndex?: number): T | undefined;
/** @internal */
export function find<T>(array: readonly T[] | undefined, predicate: (element: T, index: number) => boolean, startIndex?: number): T | undefined {
    if (array === undefined) return undefined;
    for (let i = startIndex ?? 0; i < array.length; i++) {
        const value = array[i];
        if (predicate(value, i)) {
            return value;
        }
    }
    return undefined;
}

/**
 * Works like Array.prototype.findIndex, returning `-1` if no element satisfying the predicate is found.
 *
 * @internal
 */
export function findIndex<T>(array: readonly T[] | undefined, predicate: (element: T, index: number) => boolean, startIndex?: number): number {
    if (array === undefined) return -1;
    for (let i = startIndex ?? 0; i < array.length; i++) {
        if (predicate(array[i], i)) {
            return i;
        }
    }
    return -1;
}

/** @internal */
export function contains<T>(array: readonly T[] | undefined, value: T, equalityComparer: EqualityComparer<T> = equateValues): boolean {
    if (array) {
        for (const v of array) {
            if (equalityComparer(v, value)) {
                return true;
            }
        }
    }
    return false;
}

/**
 * Filters an array by a predicate function. Returns the same array instance if the predicate is
 * true for all elements, otherwise returns a new array instance containing the filtered subset.
 *
 * @internal
 */
export function filter<T, U extends T>(array: T[], f: (x: T) => x is U): U[];
/** @internal */
export function filter<T>(array: T[], f: (x: T) => boolean): T[];
/** @internal */
export function filter<T, U extends T>(array: readonly T[], f: (x: T) => x is U): readonly U[];
/** @internal */
export function filter<T>(array: readonly T[], f: (x: T) => boolean): readonly T[];
/** @internal */
export function filter<T, U extends T>(array: T[] | undefined, f: (x: T) => x is U): U[] | undefined;
/** @internal */
export function filter<T>(array: T[] | undefined, f: (x: T) => boolean): T[] | undefined;
/** @internal */
export function filter<T, U extends T>(array: readonly T[] | undefined, f: (x: T) => x is U): readonly U[] | undefined;
/** @internal */
export function filter<T>(array: readonly T[] | undefined, f: (x: T) => boolean): readonly T[] | undefined;
/** @internal */
export function filter<T>(array: readonly T[] | undefined, f: (x: T) => boolean): readonly T[] | undefined {
    if (array) {
        const len = array.length;
        let i = 0;
        while (i < len && f(array[i])) i++;
        if (i < len) {
            const result = array.slice(0, i);
            i++;
            while (i < len) {
                const item = array[i];
                if (f(item)) {
                    result.push(item);
                }
                i++;
            }
            return result;
        }
    }
    return array;
}

/** @internal */
export function map<T, U>(array: readonly T[], f: (x: T, i: number) => U): U[];
/** @internal */
export function map<T, U>(array: readonly T[] | undefined, f: (x: T, i: number) => U): U[] | undefined;
/** @internal */
export function map<T, U>(array: readonly T[] | undefined, f: (x: T, i: number) => U): U[] | undefined {
    let result: U[] | undefined;
    if (array) {
        result = [];
        for (let i = 0; i < array.length; i++) {
            result.push(f(array[i], i));
        }
    }
    return result;
}

/**
 * Maps from T to T and avoids allocation if all elements map to themselves
 *
 * @internal */
export function sameMap<T, U = T>(array: T[], f: (x: T, i: number) => U): U[];
/** @internal */
export function sameMap<T, U = T>(array: readonly T[], f: (x: T, i: number) => U): readonly U[];
/** @internal */
export function sameMap<T, U = T>(array: T[] | undefined, f: (x: T, i: number) => U): U[] | undefined;
/** @internal */
export function sameMap<T, U = T>(array: readonly T[] | undefined, f: (x: T, i: number) => U): readonly U[] | undefined;
/** @internal */
export function sameMap<T, U = T>(array: readonly T[] | undefined, f: (x: T, i: number) => U): readonly U[] | undefined {
    if (array) {
        for (let i = 0; i < array.length; i++) {
            const item = array[i];
            const mapped = f(item, i);
            if (item as unknown !== mapped) {
                const result: U[] = array.slice(0, i) as unknown[] as U[];
                result.push(mapped);
                for (i++; i < array.length; i++) {
                    result.push(f(array[i], i));
                }
                return result;
            }
        }
    }
    return array as unknown[] as U[];
}

/**
 * Maps an array. If the mapped value is an array, it is spread into the result.
 *
 * @param array The array to map.
 * @param mapfn The callback used to map the result into one or more values.
 *
 * @internal
 */
export function flatMap<T, U>(array: readonly T[] | undefined, mapfn: (x: T, i: number) => U | readonly U[] | undefined): readonly U[] {
    let result: U[] | undefined;
    if (array) {
        for (let i = 0; i < array.length; i++) {
            const v = mapfn(array[i], i);
            if (v) {
                if (isArray(v)) {
                    result = addRange(result, v);
                }
                else {
                    result = append(result, v);
                }
            }
        }
    }
    return result || emptyArray;
}

/**
 * Maps an array. If the mapped value is an array, it is spread into the result.
 * Avoids allocation if all elements map to themselves.
 *
 * @param array The array to map.
 * @param mapfn The callback used to map the result into one or more values.
 *
 * @internal
 */
export function sameFlatMap<T>(array: T[], mapfn: (x: T, i: number) => T | readonly T[]): T[];
/** @internal */
export function sameFlatMap<T>(array: readonly T[], mapfn: (x: T, i: number) => T | readonly T[]): readonly T[];
/** @internal */
export function sameFlatMap<T>(array: readonly T[], mapfn: (x: T, i: number) => T | readonly T[]): readonly T[] {
    let result: T[] | undefined;
    if (array) {
        for (let i = 0; i < array.length; i++) {
            const item = array[i];
            const mapped = mapfn(item, i);
            if (result || item !== mapped || isArray(mapped)) {
                if (!result) {
                    result = array.slice(0, i);
                }
                if (isArray(mapped)) {
                    addRange(result, mapped);
                }
                else {
                    result.push(mapped);
                }
            }
        }
    }
    return result || array;
}

/** @internal */
export function mapDefined<T, U>(array: readonly T[] | undefined, mapFn: (x: T, i: number) => U | undefined): U[] {
    const result: U[] = [];
    if (array) {
        for (let i = 0; i < array.length; i++) {
            const mapped = mapFn(array[i], i);
            if (mapped !== undefined) {
                result.push(mapped);
            }
        }
    }
    return result;
}

/** @internal */
export function some<T>(array: readonly T[] | undefined): array is readonly T[];
/** @internal */
export function some<T>(array: readonly T[] | undefined, predicate: (value: T) => boolean): boolean;
/** @internal */
export function some<T>(array: readonly T[] | undefined, predicate?: (value: T) => boolean): boolean {
    if (array) {
        if (predicate) {
            for (const v of array) {
                if (predicate(v)) {
                    return true;
                }
            }
        }
        else {
            return array.length > 0;
        }
    }
    return false;
}

/** @internal */
export function concatenate<T>(array1: T[], array2: T[]): T[];
/** @internal */
export function concatenate<T>(array1: readonly T[], array2: readonly T[]): readonly T[];
/** @internal */
export function concatenate<T>(array1: T[], array2: T[] | undefined): T[]; // eslint-disable-line @typescript-eslint/unified-signatures
/** @internal */
export function concatenate<T>(array1: T[] | undefined, array2: T[]): T[]; // eslint-disable-line @typescript-eslint/unified-signatures
/** @internal */
export function concatenate<T>(array1: readonly T[], array2: readonly T[] | undefined): readonly T[]; // eslint-disable-line @typescript-eslint/unified-signatures
/** @internal */
export function concatenate<T>(array1: readonly T[] | undefined, array2: readonly T[]): readonly T[]; // eslint-disable-line @typescript-eslint/unified-signatures
/** @internal */
export function concatenate<T>(array1: T[] | undefined, array2: T[] | undefined): T[] | undefined;
/** @internal */
export function concatenate<T>(array1: readonly T[] | undefined, array2: readonly T[] | undefined): readonly T[] | undefined;
/** @internal */
export function concatenate<T>(array1: readonly T[] | undefined, array2: readonly T[] | undefined): readonly T[] | undefined {
    if (!some(array2)) return array1;
    if (!some(array1)) return array2;
    return [...array1, ...array2];
}

function selectIndex(_: unknown, i: number) {
    return i;
}

/** @internal */
function indicesOf(array: readonly unknown[]): number[] {
    return array.map(selectIndex);
}

/**
 * Appends a value to an array, returning the array.
 *
 * @param to The array to which `value` is to be appended. If `to` is `undefined`, a new array
 * is created if `value` was appended.
 * @param value The value to append to the array. If `value` is `undefined`, nothing is
 * appended.
 *
 * @internal
 */
export function append<TArray extends any[] | undefined, TValue extends NonNullable<TArray>[number] | undefined>(to: TArray, value: TValue): [undefined, undefined] extends [TArray, TValue] ? TArray : NonNullable<TArray>[number][];
/** @internal */
export function append<T>(to: T[], value: T | undefined): T[];
/** @internal */
export function append<T>(to: T[] | undefined, value: T): T[];
/** @internal */
export function append<T>(to: T[] | undefined, value: T | undefined): T[] | undefined;
/** @internal */
export function append<T>(to: T[], value: T | undefined): void;
/** @internal */
export function append<T>(to: T[] | undefined, value: T | undefined): T[] | undefined {
    if (value === undefined) return to as T[];
    if (to === undefined) return [value];
    to.push(value);
    return to;
}

/**
 * Gets the actual offset into an array for a relative offset. Negative offsets indicate a
 * position offset from the end of the array.
 */
function toOffset(array: readonly any[], offset: number) {
    return offset < 0 ? array.length + offset : offset;
}

/**
 * Appends a range of value to an array, returning the array.
 *
 * @param to The array to which `value` is to be appended. If `to` is `undefined`, a new array
 * is created if `value` was appended.
 * @param from The values to append to the array. If `from` is `undefined`, nothing is
 * appended. If an element of `from` is `undefined`, that element is not appended.
 * @param start The offset in `from` at which to start copying values.
 * @param end The offset in `from` at which to stop copying values (non-inclusive).
 *
 * @internal
 */
export function addRange<T>(to: T[], from: readonly T[] | undefined, start?: number, end?: number): T[];
/** @internal */
export function addRange<T>(to: T[] | undefined, from: readonly T[] | undefined, start?: number, end?: number): T[] | undefined;
/** @internal */
export function addRange<T>(to: T[] | undefined, from: readonly T[] | undefined, start?: number, end?: number): T[] | undefined {
    if (from === undefined || from.length === 0) return to;
    if (to === undefined) return from.slice(start, end);
    start = start === undefined ? 0 : toOffset(from, start);
    end = end === undefined ? from.length : toOffset(from, end);
    for (let i = start; i < end && i < from.length; i++) {
        if (from[i] !== undefined) {
            to.push(from[i]);
        }
    }
    return to;
}

/**
 * @return Whether the value was added.
 *
 * @internal
 */
function pushIfUnique<T>(array: T[], toAdd: T, equalityComparer?: EqualityComparer<T>): boolean {
    if (contains(array, toAdd, equalityComparer)) {
        return false;
    }
    else {
        array.push(toAdd);
        return true;
    }
}

/**
 * Unlike `pushIfUnique`, this can take `undefined` as an input, and returns a new array.
 *
 * @internal
 */
export function appendIfUnique<T>(array: T[] | undefined, toAdd: T, equalityComparer?: EqualityComparer<T>): T[] {
    if (array) {
        pushIfUnique(array, toAdd, equalityComparer);
        return array;
    }
    else {
        return [toAdd];
    }
}

function stableSortIndices<T>(array: readonly T[], indices: number[], comparer: Comparer<T>) {
    // sort indices by value then position
    indices.sort((x, y) => comparer(array[x], array[y]) || compareValues(x, y));
}

/**
 * Stable sort of an array. Elements equal to each other maintain their relative position in the array.
 *
 * @internal
 */
export function stableSort<T>(array: readonly T[], comparer: Comparer<T>): SortedReadonlyArray<T> {
    const indices = indicesOf(array);
    stableSortIndices(array, indices, comparer);
    return indices.map(i => array[i]) as SortedArray<T> as SortedReadonlyArray<T>;
}

/**
 * Returns the first element of an array if non-empty, `undefined` otherwise.
 *
 * @internal
 */
export function firstOrUndefined<T>(array: readonly T[] | undefined): T | undefined {
    return array === undefined || array.length === 0 ? undefined : array[0];
}

/**
 * Returns the last element of an array if non-empty, `undefined` otherwise.
 *
 * @internal
 */
export function lastOrUndefined<T>(array: readonly T[] | undefined): T | undefined {
    return array === undefined || array.length === 0 ? undefined : array[array.length - 1];
}

/** @internal */
export function last<T>(array: readonly T[]): T {
    Debug.assert(array.length !== 0);
    return array[array.length - 1];
}

/**
 * Returns the only element of an array if it contains only one element, `undefined` otherwise.
 *
 * @internal
 */
export function singleOrUndefined<T>(array: readonly T[] | undefined): T | undefined {
    return array && array.length === 1
        ? array[0]
        : undefined;
}

/**
 * Performs a binary search, finding the index at which `value` occurs in `array`.
 * If no such index is found, returns the 2's-complement of first index at which
 * `array[index]` exceeds `value`.
 * @param array A sorted array whose first element must be no larger than number
 * @param value The value to be searched for in the array.
 * @param keySelector A callback used to select the search key from `value` and each element of
 * `array`.
 * @param keyComparer A callback used to compare two keys in a sorted array.
 * @param offset An offset into `array` at which to start the search.
 *
 * @internal
 */
export function binarySearch<T, U>(array: readonly T[], value: T, keySelector: (v: T) => U, keyComparer: Comparer<U>, offset?: number): number {
    return binarySearchKey(array, keySelector(value), keySelector, keyComparer, offset);
}

/**
 * Performs a binary search, finding the index at which an object with `key` occurs in `array`.
 * If no such index is found, returns the 2's-complement of first index at which
 * `array[index]` exceeds `key`.
 * @param array A sorted array whose first element must be no larger than number
 * @param key The key to be searched for in the array.
 * @param keySelector A callback used to select the search key from each element of `array`.
 * @param keyComparer A callback used to compare two keys in a sorted array.
 * @param offset An offset into `array` at which to start the search.
 *
 * @internal
 */
function binarySearchKey<T, U>(array: readonly T[], key: U, keySelector: (v: T, i: number) => U, keyComparer: Comparer<U>, offset?: number): number {
    if (!some(array)) {
        return -1;
    }

    let low = offset || 0;
    let high = array.length - 1;
    while (low <= high) {
        const middle = low + ((high - low) >> 1);
        const midKey = keySelector(array[middle], middle);
        switch (keyComparer(midKey, key)) {
            case Comparison.LessThan:
                low = middle + 1;
                break;
            case Comparison.EqualTo:
                return middle;
            case Comparison.GreaterThan:
                high = middle - 1;
                break;
        }
    }

    return ~low;
}

/** @internal */
export function reduceLeft<T, U>(array: readonly T[] | undefined, f: (memo: U, value: T, i: number) => U, initial: U, start?: number, count?: number): U;
/** @internal */
export function reduceLeft<T>(array: readonly T[], f: (memo: T, value: T, i: number) => T): T | undefined;
/** @internal */
export function reduceLeft<T>(array: readonly T[] | undefined, f: (memo: T, value: T, i: number) => T, initial?: T, start?: number, count?: number): T | undefined {
    if (array && array.length > 0) {
        const size = array.length;
        if (size > 0) {
            let pos = start === undefined || start < 0 ? 0 : start;
            const end = count === undefined || pos + count > size - 1 ? size - 1 : pos + count;
            let result: T;
            if (arguments.length <= 2) {
                result = array[pos];
                pos++;
            }
            else {
                result = initial!;
            }
            while (pos <= end) {
                result = f(result, array[pos], pos);
                pos++;
            }
            return result;
        }
    }
    return initial;
}

const hasOwnProperty = Object.prototype.hasOwnProperty;

/**
 * Indicates whether a map-like contains an own property with the specified key.
 *
 * @param map A map-like.
 * @param key A property key.
 *
 * @internal
 */
export function hasProperty(map: MapLike<any>, key: string): boolean {
    return hasOwnProperty.call(map, key);
}

/**
 * Gets the owned, enumerable property keys of a map-like.
 *
 * @internal
 */
export function getOwnKeys<T>(map: MapLike<T>): string[] {
    const keys: string[] = [];
    for (const key in map) {
        if (hasOwnProperty.call(map, key)) {
            keys.push(key);
        }
    }

    return keys;
}

/**
 * Tests whether a value is an array.
 *
 * @internal
 */
export function isArray(value: any): value is readonly unknown[] {
    // See: https://github.com/microsoft/TypeScript/issues/17002
    return Array.isArray(value);
}

/** @internal */
export function toArray<T>(value: T | T[]): T[];
/** @internal */
export function toArray<T>(value: T | readonly T[]): readonly T[];
/** @internal */
export function toArray<T>(value: T | T[]): T[] {
    return isArray(value) ? value : [value];
}

/**
 * Tests whether a value is string
 *
 * @internal
 */
export function isString(text: unknown): text is string {
    return typeof text === "string";
}

/** @internal */
export function cast<TOut extends TIn, TIn = any>(value: TIn | undefined, test: (value: TIn) => value is TOut): TOut {
    if (value !== undefined && test(value)) return value;

    return Debug.fail(`Invalid cast. The supplied value ${value} did not pass the test '${Debug.getFunctionName(test)}'.`);
}

/**
 * Does nothing.
 *
 * @internal
 */
export function noop(_?: unknown): void {}

/**
 * Do nothing and return true
 *
 * @internal
 */
export function returnTrue(): true {
    return true;
}

/**
 * Returns its argument.
 *
 * @internal
 */
export function identity<T>(x: T) {
    return x;
}

// We convert the file names to lower case as key for file name on case insensitive file system
// While doing so we need to handle special characters (eg \u0130) to ensure that we dont convert
// it to lower case, fileName with its lowercase form can exist along side it.
// Handle special characters and make those case sensitive instead
//
// |-#--|-Unicode--|-Char code-|-Desc-------------------------------------------------------------------|
// | 1. | i        | 105       | Ascii i                                                                |
// | 2. | I        | 73        | Ascii I                                                                |
// |-------- Special characters ------------------------------------------------------------------------|
// | 3. | \u0130   | 304       | Upper case I with dot above                                            |
// | 4. | i,\u0307 | 105,775   | i, followed by 775: Lower case of (3rd item)                           |
// | 5. | I,\u0307 | 73,775    | I, followed by 775: Upper case of (4th item), lower case is (4th item) |
// | 6. | \u0131   | 305       | Lower case i without dot, upper case is I (2nd item)                   |
// | 7. | \u00DF   | 223       | Lower case sharp s                                                     |
//
// Because item 3 is special where in its lowercase character has its own
// upper case form we cant convert its case.
// Rest special characters are either already in lower case format or
// they have corresponding upper case character so they dont need special handling
//
// But to avoid having to do string building for most common cases, also ignore
// a-z, 0-9, \u0131, \u00DF, \, /, ., : and space

/** @internal */
export function memoize<T>(callback: () => T): () => T {
    let value: T;
    return () => {
        if (callback) {
            value = callback();
            callback = undefined!;
        }
        return value;
    };
}

/**
 * A version of `memoize` that supports a single primitive argument
 *
 * @internal
 */
export function memoizeOne<A extends string | number | boolean | undefined, T>(callback: (arg: A) => T): (arg: A) => T {
    const map = new Map<string, T>();
    return (arg: A) => {
        const key = `${typeof arg}:${arg}`;
        let value = map.get(key);
        if (value === undefined && !map.has(key)) {
            value = callback(arg);
            map.set(key, value);
        }
        return value!;
    };
}

/** @internal */
export const enum AssertionLevel {
    None = 0,
    Normal = 1,
    Aggressive = 2,
    VeryAggressive = 3,
}

/**
 * Safer version of `Function` which should not be called.
 * Every function should be assignable to this, but this should not be assignable to every function.
 *
 * @internal
 */
export type AnyFunction = (...args: never[]) => void;

/** @internal */
function equateValues<T>(a: T, b: T) {
    return a === b;
}

/**
 * Compare the equality of two strings using a case-sensitive ordinal comparison.
 *
 * Case-sensitive comparisons compare both strings one code-point at a time using the integer
 * value of each code-point after applying `toUpperCase` to each string. We always map both
 * strings to their upper-case form as some unicode characters do not properly round-trip to
 * lowercase (such as `ẞ` (German sharp capital s)).
 *
 * @internal
 */
export function equateStringsCaseInsensitive(a: string, b: string) {
    return a === b
        || a !== undefined
            && b !== undefined
            && a.toUpperCase() === b.toUpperCase();
}

/**
 * Compare the equality of two strings using a case-sensitive ordinal comparison.
 *
 * Case-sensitive comparisons compare both strings one code-point at a time using the
 * integer value of each code-point.
 *
 * @internal
 */
export function equateStringsCaseSensitive(a: string, b: string) {
    return equateValues(a, b);
}

function compareComparableValues(a: string | undefined, b: string | undefined): Comparison;
function compareComparableValues(a: number | undefined, b: number | undefined): Comparison;
function compareComparableValues(a: string | number | undefined, b: string | number | undefined) {
    return a === b ? Comparison.EqualTo :
        a === undefined ? Comparison.LessThan :
        b === undefined ? Comparison.GreaterThan :
        a < b ? Comparison.LessThan :
        Comparison.GreaterThan;
}

/**
 * Compare two numeric values for their order relative to each other.
 * To compare strings, use any of the `compareStrings` functions.
 *
 * @internal
 */
export function compareValues(a: number | undefined, b: number | undefined): Comparison {
    return compareComparableValues(a, b);
}

/**
 * Given a name and a list of names that are *not* equal to the name, return a spelling suggestion if there is one that is close enough.
 * Names less than length 3 only check for case-insensitive equality.
 *
 * find the candidate with the smallest Levenshtein distance,
 *    except for candidates:
 *      * With no name
 *      * Whose length differs from the target name by more than 0.34 of the length of the name.
 *      * Whose levenshtein distance is more than 0.4 of the length of the name
 *        (0.4 allows 1 substitution/transposition for every 5 characters,
 *         and 1 insertion/deletion at 3 characters)
 *
 * @internal
 */
export function getSpellingSuggestion<T>(name: string, candidates: T[], getName: (candidate: T) => string | undefined): T | undefined {
    const maximumLengthDifference = Math.max(2, Math.floor(name.length * 0.34));
    let bestDistance = Math.floor(name.length * 0.4) + 1; // If the best result is worse than this, don't bother.
    let bestCandidate: T | undefined;
    for (const candidate of candidates) {
        const candidateName = getName(candidate);
        if (candidateName !== undefined && Math.abs(candidateName.length - name.length) <= maximumLengthDifference) {
            if (candidateName === name) {
                continue;
            }
            // Only consider candidates less than 3 characters long when they differ by case.
            // Otherwise, don't bother, since a user would usually notice differences of a 2-character name.
            if (candidateName.length < 3 && candidateName.toLowerCase() !== name.toLowerCase()) {
                continue;
            }

            const distance = levenshteinWithMax(name, candidateName, bestDistance - 0.1);
            if (distance === undefined) {
                continue;
            }

            Debug.assert(distance < bestDistance); // Else `levenshteinWithMax` should return undefined
            bestDistance = distance;
            bestCandidate = candidate;
        }
    }
    return bestCandidate;
}

function levenshteinWithMax(s1: string, s2: string, max: number): number | undefined {
    let previous = new Array(s2.length + 1);
    let current = new Array(s2.length + 1);
    /** Represents any value > max. We don't care about the particular value. */
    const big = max + 0.01;

    for (let i = 0; i <= s2.length; i++) {
        previous[i] = i;
    }

    for (let i = 1; i <= s1.length; i++) {
        const c1 = s1.charCodeAt(i - 1);
        const minJ = Math.ceil(i > max ? i - max : 1);
        const maxJ = Math.floor(s2.length > max + i ? max + i : s2.length);
        current[0] = i;
        /** Smallest value of the matrix in the ith column. */
        let colMin = i;
        for (let j = 1; j < minJ; j++) {
            current[j] = big;
        }
        for (let j = minJ; j <= maxJ; j++) {
            // case difference should be significantly cheaper than other differences
            const substitutionDistance = s1[i - 1].toLowerCase() === s2[j - 1].toLowerCase()
                ? (previous[j - 1] + 0.1)
                : (previous[j - 1] + 2);
            const dist = c1 === s2.charCodeAt(j - 1)
                ? previous[j - 1]
                : Math.min(/*delete*/ previous[j] + 1, /*insert*/ current[j - 1] + 1, /*substitute*/ substitutionDistance);
            current[j] = dist;
            colMin = Math.min(colMin, dist);
        }
        for (let j = maxJ + 1; j <= s2.length; j++) {
            current[j] = big;
        }
        if (colMin > max) {
            // Give up -- everything in this column is > max and it can't get better in future columns.
            return undefined;
        }

        const temp = previous;
        previous = current;
        current = temp;
    }

    const res = previous[s2.length];
    return res > max ? undefined : res;
}

/** @internal */
export function endsWith(str: string, suffix: string, ignoreCase?: boolean): boolean {
    const expectedPos = str.length - suffix.length;
    return expectedPos >= 0 && (
        ignoreCase
            ? equateStringsCaseInsensitive(str.slice(expectedPos), suffix)
            : str.indexOf(suffix, expectedPos) === expectedPos
    );
}

/**
 * Represents a "prefix*suffix" pattern.
 *
 * @internal
 */
export interface Pattern {
    prefix: string;
    suffix: string;
}

/** @internal */
export function startsWith(str: string, prefix: string, ignoreCase?: boolean): boolean {
    return ignoreCase
        ? equateStringsCaseInsensitive(str.slice(0, prefix.length), prefix)
        : str.lastIndexOf(prefix, 0) === 0;
}
