// @noEmit: true
// @noTypesAndSymbols: true

type MaybeBaseType = any;
interface Flavoring<FlavorT> {
    _type?: FlavorT;
}
type Flavor<T, FlavorT> = T & Flavoring<FlavorT>;
type IdOf<T> = T extends {
    id: infer I | undefined;
}
    ? I
    : never;

type ValueFilter<V, N> =
    | V
    | V[]
    | N
    | undefined
    | {
        eq: V | N | undefined;
    }
    | {
        ne: V | N | undefined;
    }
    | {
        in: (V | N)[] | undefined;
    }
    | {
        nin: (V | N)[] | undefined;
    }
    | {
        gt: V | undefined;
    }
    | {
        gte: V | undefined;
    }
    | {
        lt: V | undefined;
    }
    | {
        lte: V | undefined;
    }
    | {
        like: V | undefined;
    }
    | {
        ilike: V | undefined;
    }
    | {
        contains: V | undefined;
    }
    | {
        overlaps: V | undefined;
    }
    | {
        containedBy: V | undefined;
    }
    | {
        gte: V | undefined;
        lte: V | undefined;
    }
    | {
        between: [V, V] | undefined;
    };

interface Collection<T extends Entity, U extends Entity>
    extends Relation<T, U> {
    load(opts?: { withDeleted: boolean }): Promise<ReadonlyArray<U>>;
    find(id: IdOf<U>): Promise<U | undefined>;
    includes(other: U): Promise<boolean>;
    add(other: U): void;
    remove(other: U): void;
    readonly isLoaded: boolean;
}

type SuffixSeperator = ":" | "_";
type DropSuffix<K> = K extends `${infer key}${SuffixSeperator}ro` ? key : K;
type NormalizeHint<T extends Entity, H> = H extends string
    ? Record<DropSuffix<H>, {}>
    : H extends ReadonlyArray<any>
    ? Record<DropSuffix<H[number]>, {}>
    : {
        [K in keyof H as DropSuffix<K>]: H[K];
    };
declare const I: unique symbol;
interface AsyncProperty<T extends Entity, V> {
    isLoaded: boolean;
    load(): Promise<V>;
    [I]?: T;
}
type LoadableValue<V> = V extends Reference<any, infer U, any>
    ? U
    : V extends Collection<any, infer U>
    ? U
    : V extends AsyncProperty<any, infer P>
    ? P extends infer U extends Entity | undefined
    ? U
    : P
    : never;
type Loadable<T extends Entity> = {
    -readonly [K in keyof T as LoadableValue<T[K]> extends never
        ? never
        : K]: LoadableValue<T[K]>;
};
interface LoadedReference<
    T extends Entity,
    U extends Entity,
    N extends never | undefined,
> extends Reference<T, U, N> {
    id: IdOf<U> | N;
    get: U | N;
    getWithDeleted: U | N;
    isSet: boolean;
}
interface LoadedOneToOneReference<T extends Entity, U extends Entity>
    extends LoadedReference<T, U, undefined> {
    get: U | undefined;
    getWithDeleted: U | undefined;
    idOrFail: IdOf<U>;
    idUntagged: string | undefined;
    idUntaggedOrFail: string;
    readonly isSet: boolean;
}
interface LoadedCollection<T extends Entity, U extends Entity>
    extends Collection<T, U> {
    get: ReadonlyArray<U>;
    getWithDeleted: ReadonlyArray<U>;
    set(values: U[]): void;
    removeAll(): void;
}
interface LoadedProperty<T extends Entity, V> {
    get: V;
}

type LoadHint<T extends Entity> =
    | (keyof Loadable<T> & string)
    | ReadonlyArray<keyof Loadable<T> & string>
    | NestedLoadHint<T>;

type NestedLoadHint<T extends Entity> = {
    [K in keyof Loadable<T>]?: Loadable<T>[K] extends infer U extends Entity
        ? LoadHint<U>
        : {};
};

declare const deepLoad: unique symbol;
type DeepLoadHint<T extends Entity> = NestedLoadHint<T> & {
    [deepLoad]: true;
};

type MarkDeepLoaded<T extends Entity, P> = P extends OneToOneReference<
    MaybeBaseType,
    infer U
>
    ? LoadedOneToOneReference<T, Loaded<U, DeepLoadHint<U>>>
    : P extends Reference<MaybeBaseType, infer U, infer N>
    ? LoadedReference<T, Loaded<U, DeepLoadHint<U>>, N>
    : P extends Collection<MaybeBaseType, infer U>
    ? LoadedCollection<T, Loaded<U, DeepLoadHint<U>>>
    : P extends AsyncProperty<MaybeBaseType, infer V>
    ? [V] extends [(infer U extends Entity) | undefined]
    ? LoadedProperty<T, Loaded<U, DeepLoadHint<U>> | Exclude<V, U>>
    : V extends readonly (infer U extends Entity)[]
    ? LoadedProperty<T, Loaded<U, DeepLoadHint<U>>[]>
    : LoadedProperty<T, V>
    : unknown;

type MarkLoaded<T extends Entity, P, UH = {}> = P extends OneToOneReference<
    MaybeBaseType,
    infer U
>
    ? LoadedOneToOneReference<T, Loaded<U, UH>>
    : P extends Reference<MaybeBaseType, infer U, infer N>
    ? LoadedReference<T, Loaded<U, UH>, N>
    : P extends Collection<MaybeBaseType, infer U>
    ? LoadedCollection<T, Loaded<U, UH>>
    : P extends AsyncProperty<MaybeBaseType, infer V>
    ? [V] extends [(infer U extends Entity) | undefined]
    ? LoadedProperty<T, Loaded<U, UH> | Exclude<V, U>>
    : V extends readonly (infer U extends Entity)[]
    ? LoadedProperty<T, Loaded<U, UH>[]>
    : LoadedProperty<T, V>
    : unknown;

type Loaded<T extends Entity, H> = T & {
    [K in keyof T & keyof NormalizeHint<T, H>]: H extends DeepLoadHint<T>
        ? MarkDeepLoaded<T, T[K]>
        : MarkLoaded<T, T[K], NormalizeHint<T, H>[K]>;
};

interface Entity {}

declare const RelationT: unique symbol;
declare const RelationU: unique symbol;
interface Relation<T extends Entity, U extends Entity> {
    [RelationT]: T;
    [RelationU]: U;
    isLoaded: boolean;
}

declare const ReferenceN: unique symbol;
interface Reference<
    T extends Entity,
    U extends Entity,
    N extends never | undefined,
> extends Relation<T, U> {
    readonly isLoaded: boolean;
    load(opts?: { withDeleted?: boolean; forceReload?: true }): Promise<U | N>;
    set(other: U | N): void;
    [ReferenceN]: N;
}
declare const OneToOne: unique symbol;
interface OneToOneReference<T extends Entity, U extends Entity>
    extends Reference<T, U, undefined> {
    [OneToOne]: T;
}

interface PolymorphicReference<
    T extends Entity,
    U extends Entity,
    N extends never | undefined,
> extends Reference<T, U, N> {
    id: IdOf<U> | undefined;
    idOrFail: IdOf<U>;
    idUntagged: string | undefined;
    idUntaggedOrFail: string;
    readonly isSet: boolean;
}

type BidContractId = Flavor<string, BidContract>;

interface BidContractFilter {
    id?: ValueFilter<BidContractId, never>;
}

declare abstract class BidContractCodegen {
    static readonly tagName = "bc";

    readonly __orm: {
        filterType: BidContractFilter;
    };

    readonly collaboration: OneToOneReference<BidContract, Collaboration>;
}

class BidContract extends BidContractCodegen {}

type BidItemId = Flavor<string, BidItem>;

interface BidItemFilter {
    id?: ValueFilter<BidItemId, never>;
}

declare abstract class BidItemCodegen {
    static readonly tagName = "bi";

    readonly __orm: {
        filterType: BidItemFilter;
    };

    readonly collaboration: OneToOneReference<BidItem, Collaboration>;
}

declare class BidItem extends BidItemCodegen {}

type BillId = Flavor<string, Bill>;

interface BillFilter {
    id?: ValueFilter<BillId, never>;
}

declare abstract class BillCodegen {
    static readonly tagName = "b";

    readonly __orm: {
        filterType: BillFilter;
    };

    readonly collaboration: OneToOneReference<Bill, Collaboration>;
}

declare class Bill extends BillCodegen {}

export type CollaborationParent = BidContract | BidItem | Bill;

declare abstract class CollaborationCodegen {
    static readonly tagName = "collab";
    readonly parent: PolymorphicReference<
        Collaboration,
        CollaborationParent,
        never
    >;
}

class Collaboration extends CollaborationCodegen {}

type LoadedCollaboration = Loaded<Collaboration, "parent">;

type CollaborationMetadata = {
    id: string;
    projectId: string | undefined;
    name: string;
    blueprintPath: string;
    type: string;
};

declare function getCollaborationMetadata(
    c: LoadedCollaboration,
): Promise<CollaborationMetadata>;

declare const collaboration: Loaded<
    Collaboration,
    DeepLoadHint<Collaboration>
>;

getCollaborationMetadata(collaboration);

export {};
