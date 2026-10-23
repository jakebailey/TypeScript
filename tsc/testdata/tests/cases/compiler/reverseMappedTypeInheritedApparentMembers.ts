// @strict: true
// @noEmit: true

// Inherited members of an interface whose base is a type parameter instantiated
// with a primitive come from the apparent type (here, String). Reverse mapped
// type inference must still be able to resolve them.

declare const Base: new <T extends object>() => T;
// @ts-ignore
declare class Wrapper<T extends object> extends Base<T> {}
interface Derived<T extends {}> extends Wrapper<T> {}

type Mapped<T> = { [K in keyof T]: Derived<K> };
declare function f<T>(arg: Mapped<T>): void;

f({ a: {} });
