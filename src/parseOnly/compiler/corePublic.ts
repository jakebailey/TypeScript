// WARNING: The script `configurePrerelease.ts` uses a regexp to parse out these values.
// If changing the text in this section, be sure to test `configurePrerelease` too.
// The following is baselined as a literal template type without intervention

/**
 * Type of objects whose values are all of the same type.
 * The `in` and `for-in` operators can *not* be safely used,
 * since `Object.prototype` may be modified by outside code.
 */
export interface MapLike<T> {
    [index: string]: T;
}

export interface SortedReadonlyArray<T> extends ReadonlyArray<T> {
    " __sortedArrayBrand": any;
}

export interface SortedArray<T> extends Array<T> {
    " __sortedArrayBrand": any;
}

/** @internal */
export type EqualityComparer<T> = (a: T, b: T) => boolean;

/** @internal */
export type Comparer<T> = (a: T, b: T) => Comparison;

/** @internal */
export const enum Comparison {
    LessThan = -1,
    EqualTo = 0,
    GreaterThan = 1,
}
