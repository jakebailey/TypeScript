# Arena AST: nodes as offsets into per-file, pointer-free storage

Branch: [`jakebailey/TypeScript@arena-ast`](https://github.com/jakebailey/TypeScript/tree/arena-ast) (previous attempt backed up at `arena-ast-old`). This file is the living design doc for the branch and is updated alongside it.

## Goal

Every `*ast.Node` is a heap object full of Go pointers: children, parent, symbol, flow node, lists, and so on. The GC has to trace all of them. The idea here is that a node is "a file plus an offset", and its fields are pointer-free data, so the GC barely sees the AST.

## Design

A node is the address of a pointer-free record in a per-file chunk of words. Fields refer to other records by 32-bit refs, mostly as offsets. The garbage collector sees one pointer per chunk, not one per child, parent, symbol and string.

### A node is a record address

```go
type Node struct{ p unsafe.Pointer }        // address of the node's record
type PropertyAccessExpression struct{ Node } // typed views carry no extra data
```

`Node` is one word, so it is passed in a register and is cheap to compare and to use as a map key. The zero `Node` is "no node". Kind-specific views are free conversions. `Kind()` is one load from the record header.

### Records live in chunks that never move

```go
type chunkHeader struct {
    store   *nodeStore
    ids     atomic.Pointer[uint32] // node ids, see below
    index   uint32
    idCount uint16
    idCap   uint16
}
// A chunk is reflect.StructOf{Header chunkHeader; Words [n]uint64}.
```

Each store (one per `NodeFactory`, so one per parsed file) allocates records from chunks. Each chunk is a single Go object whose only pointers are in its header, so the GC never scans the records. Chunks are sized to allocator size classes (2^k or 3·2^(k-1) bytes, including the allocator's 8-byte malloc header or a pad word of our own), so no rounding is wasted. A parsed file's first chunk is sized from its source length, and later chunks add 1/16 of the store's size, up to 512KB. Records never move. An interior `Node` pointer keeps its chunk alive, and the chunk header keeps the store alive.

`ParseSourceFile` parses, binds right away while the file is still private, then freezes the store and clips its side tables. Allocating in a frozen store panics.

### Every record starts with a 20-byte header

```go
type nodeBaseData struct {
    loc    core.TextRange
    parent parentRef
    flags  NodeFlags
    kind   atomic.Uint32 // Kind (9 bits), chunk code (7 bits), id number (16 bits)
}
```

The chunk code leads from any record back to its chunk header, and from there to the store and its side tables. Every chunk header sits 8 bytes into a block aligned to 256 bytes (chunks up to 16KB) or 8KB (larger chunks): the Go allocator puts small objects in slots aligned to a power of two dividing their size class, and an 8-byte malloc header first in those with pointers over 512 bytes, and it page-aligns large objects, which get a pad word instead. So the code holds the block size (1 bit) and how many blocks back the chunk's block is (6 bits), and `chunkOf` is a mask and an add from a 128-entry table. `addChunk` panics if an allocation breaks these assumptions.

### Node ids live in the chunk

Records carry no 8-byte id. Ids are still given lazily, on first `GetNodeId`, so only the nodes that ask get one (about 7M of 20M on vscode), and they come from the same per-link-store `NodeIdGenerator` blocks as before, so the paged node link stores stay as dense as they were. The first time a node gets an id, it is appended to its chunk's `ids` table, and its 1-based position in that table goes in the high 16 bits of `kind`. Reading an id is a load of `kind`, then a load from the table. The kind word is atomic because ids are given while other goroutines read kinds.

A position-derived ordinal (record index in the store plus a per-store base) was tried and rejected: it gives every node an id, so link pages get allocated over nearly every 256-node window (about +330MB of pages on vscode). Ids shared across stores in small global blocks interleaved the link stores and cost about +73MB of pages.

### Records are flat and generated

Records are generated from the AST schema (`tools/scripts/tsc/generate-go-ast.ts`). All fields, including those of "base" shapes (declaration, function-like, flow container, …), are laid out inline after the header and sorted by alignment, so there is no padding. Accessors for members that many kinds share (`Name`, `Modifiers`, `Symbol`, `Locals`, `Body`, …) find their field through a generated per-kind offset table (`unsafe.Offsetof`). Kind-specific accessors use fixed offsets.

### How fields reference things

| Field type | Stored as | Read path |
|---|---|---|
| Node in the same chunk | `nodeRef` with the low bit set: signed byte offset from the field | `f + (r-1)` |
| No node | `nodeRef` 0 | tested before anything else |
| Node in another chunk or store | `nodeRef` even, nonzero: index into the store's `far []Node` | `store.far[r>>1]` |
| Parent | `parentRef`: even = byte offset from the record, odd = far slot `<<1\|1`, `1` = none | one add on the near path |
| Node list, modifier list | `listRef`, same encoding as `nodeRef`, over `farLists` | as for nodes |
| Flow node, flow list | index into the store's `flows`/`flowLists` arrays (pointer-free, 16 bytes each) | `FlowNode{store, i}` |
| `*Symbol`, `SymbolTable`, `[]string`, `[]Node`, `any` | `sideRef`: index into a typed side table, whose slot 0 is the zero value | `store.T.items[r]` |
| `SymbolName` | `nameRef`: an interned name's ID below 2^31 as is; any other name (derived names) is an index into the store's `names`, with the top bit set | one test, then the ID |
| `string` | `textRef{off, len}`: the top 5 bits pick one of 32 pieces of up to 128MB. The first pieces hold the source text, so source substrings are referenced in place; the rest are an append-only arena. | `unsafe.String(pieces[off>>27]+off&mask, len)` |

Lists are records too: an 8-byte header (chunk offset and layout flags, count), the packed element refs, then only those of pos, end and an aux word (missing list, or a `ModifierList`'s flags) that the list can't derive. A parsed list (`NewNodeListAt`) almost always spans exactly its elements, so it reads its pos from the first element and its end from the last; factory lists store both, since callers move their elements afterwards. A list that later changes in a way its record has no room for spills its loc and aux to a per-store map. The parser collects elements on one scratch stack and copies each finished list into the store, so parsing allocates no `[]Node` per list.

Records, lists and refs are 4-byte aligned (no record field is wider than 4 bytes), so a distance between them is never odd, and the low bit is free to mark near refs. Go's compiler can't emit a three-operand LEA, so every encoding is chosen to make the hot decode one add: near refs keep the bit set and subtract a constant, and parents use even offsets from the record itself.

### What unsafe is left

- `unsafe.Add` from a ref field to a record in the same chunk. This stays within one Go object, so it satisfies the `unsafe.Pointer` rules and `-d=checkptr`.
- Record casts `(*T)(p)`, where every `T` is pointer-free.
- Offset tables built with `unsafe.Offsetof`. The generator produces them, so nothing is laid out by hand.
- `unsafe.String` for text refs.

No Go pointer is ever stored in an integer. Every cross-chunk or cross-store reference goes through a real `[]Node` slot that the GC scans.

## Where it stands (vscode, n=8, interleaved, HEAD vs main)

```
       │    main    │            arena-ast            │
       │ check-sec  │ check-sec   vs base             │
VSCode   7.188 ± 3%   7.247 ± 3%  ~ (p=0.234 n=8)

       │    main    │            arena-ast            │
       │ total-sec  │ total-sec   vs base             │
VSCode   8.185 ± 3%   8.057 ± 2%  ~ (p=0.488 n=8)

       │     main     │             arena-ast              │
       │    mem-B     │    mem-B      vs base              │
VSCode   3.302Gi ± 0%   3.212Gi ± 0%  -2.74% (p=0.000 n=8)
```

Single-threaded with GC off, this branch executes about 8% more instructions than main, but task time is within noise (+1.3%, p=0.13), since parsing is much cheaper and the extra instructions are well-predicted branches.

### Where the remaining instructions go

- **Following a parent** costs about 6 instructions vs 1 on main: load the ref, test near/far, add, plus the far path's code size, which keeps some callers just over the inlining budget (80). Main inlines many parent-walk helpers (`FindAncestor` closures, `getControlFlowContainer`, `GetRootDeclaration`) that this branch can't. This is inherent to storing no pointers. Cross-chunk deltas would break the unsafe rules.
- **Far refs are common**: on vscode, 1.9M far node refs, 1.0M of them parents, 113K far list refs, about 6 chunks per store. A one-chunk-per-file experiment (sizing the first chunk at 2× the source) removed nearly all of them but gained only −0.5% instructions and −0.8% time, so relocating into a single chunk isn't worth its memory.
- **Shared-member accessors** read a per-kind offset table, where main reads a fixed field of the embedded base.

Experiments that didn't pay: a zero check in side-table getters (+0.03% instructions), a cached source file in the chunk header (no inlining change), early-outs in `GetRootDeclaration` (−0.09%).

### How we got here (commit sequence)

1. Value-handle `Node` codemod (`*Node` → `Node`) across the compiler.
2. Store-backed records, side tables for pointer fields, and text refs.
3. Arena chunks with a noscan body.
4. Self-relative 32-bit refs with a far-ref table, which cost 6–8% of check time at the time.
5. **Absolute address refs** (check −5.59%, memory +4.85%). These hid pointers in `uintptr`s and were later removed.
6. **Slower chunk growth** (memory −2.63%).
7. **32-byte header** (memory −3.28%).
8. **Flattened records + offset tables** (memory −0.39%).
9. **Flow nodes in the arena** (memory −0.67%).
10. **32-bit block-indexed refs** (check +10.2%, memory −1.0%), which replaced the hidden pointers with real 32-bit refs.
11. **Field-relative refs** (check +6.3% vs step 9).
12. **Index-based store** (`Node = {store, index}`, check +5.5% vs step 9).
13. **Bind at parse, then freeze.**
14. **Flow nodes as indexed values** (memory −2.9%).
15. **Lists as records** (memory −3.2%).
16. **Iterate lists in place** (check −1.0%).
17. **Kind cached in the handle** (check −1.7%). At this point check was +13% vs main and memory −4.1%. Hardware counters showed +29% instructions, all in ref-following accessors.
18. **Source file from the store** (check −1.8%).
19. **Record-address handles with self-relative refs** (`ba2d083f1da`, `2e5b2083626`). `Node` becomes a single pointer to a record in a never-moving chunk, and near refs are offsets from the field. This drops the 16–24 byte handle, which spilled around every call.
20. **Size-class chunks** (`4b85ba3d4f5`): no allocator rounding waste.
21. **Per-kind offset tables for shared members** (`ca2bc7e2fb7`).
22. **Ref encoding flip** (`a4fca73683b`): near refs carry the low bit, so the near decode is one add and parent walks inline.
23. **Accessors within the inlining budget** (`b65f487e501`).
24. **One text path** (`f6245c449b5`): source text and arena text are both pieces of one table, so `Text()` has no branch.
25. **Parent as a record-relative delta** (`86be7af1fdd`, −0.32% instructions).
26. **Zero refs tested first** (`1e5f65ab9fc`, −0.21% instructions): absent fields no longer load the far table.
27. **`GetNodeId` inlinable again** (`67837fcf02d`, −0.28% instructions).
28. **24-byte header** (vscode memory used −140MB, −3.9%): ids move from the header to per-chunk tables. Chunks shrink by 174MB, and the id tables take 35MB.
29. **4-byte names in records** (vscode memory used −61MB, −1.8%): `Identifier` keeps its `SymbolName` as a 32-bit `nameRef`, shrinking from 40 to 32 bytes. On vscode every one of the 6.9M identifier names is a plain interned ID (the largest is about 2^20), so the store's `names` table stays empty.
30. **4-byte record alignment** (vscode memory used −32MB, −1.0%): records and lists are allocated in 4-byte units instead of words, so the many records with a 4-byte tail (`PropertyAccessExpression` 48 → 44, `StringLiteral` 40 → 36, `ThisKeyword` 32 → 28, `BinaryExpression` and `PropertyAssignment` 56 → 52) no longer pad. Anchors count 4-byte units.
31. **Smaller list headers** (memory used −44MB vscode, −17MB sentry, −4MB material-ui; check time within noise): the list header shrinks from 20 bytes to 8. On vscode 3.6M lists hold only 6.2M elements, and 99.98% of non-empty parsed lists start at their first element and 96% end at their last, so pos and end are derived instead of stored, and aux is stored only when nonzero.
32. **Chunks sized for the malloc header** (memory used −50MB vscode, −29MB sentry, −11MB material-ui; check time within noise): since Go 1.22 the allocator puts an 8-byte type header before every small object with pointers larger than 512 bytes. A chunk starts with pointers, so a chunk of exactly a size class's bytes, as `chunkClass` asked for, spilled into the next class: a 4KB chunk took 4.75KB and an 8KB chunk 9.25KB. Chunks between 512 bytes and 32KB now leave room for that header.
33. **20-byte header** (memory used −87MB vscode, −35MB sentry, −11MB material-ui; check time vscode +4.9% (p=0.15), sentry −0.2%, material-ui −5.1%, so possibly a small cost on vscode): the record header drops its 4-byte chunk offset for a 7-bit chunk code packed into the kind word (Kind needs only 9 bits), found from the record's address and the alignment of chunks. Every node record shrinks by 4 bytes, and lists keep the code in their info word. Chunks are capped at 512KB so that 6 bits count their 8KB blocks.

## Next

- Find more places where the store can answer structurally what main answers by walking parents (as `GetSourceFileOfNode` does).
- Symbols as per-store typed arrays addressed by index, which would shrink the remaining GC-scanned side tables.

## Redesign: nodes as (store, index)

Branch `new-ast-index`, from `new-ast` at `1fce2af19c3`. In the design above, a node is a raw address, so everything about it has to be recovered from that address: its chunk (alignment tricks and chunk codes), its store, its file, and its id (per-chunk id tables). The redesign identifies a node by its store and its index in that store's node table, like an index into an array. Everything else hangs off the store.

### Target layout

The goal is to match a fixed reference layout, so that memory compares like for like. Deviations are listed below, with the reason Go forces each one.

- **Node header, exactly 24 bytes:** `{kind uint16, spare uint16, flags uint32, pos int32, end int32, parent uint32, word uint32}`.
  - `parent` is a node index, with 0 for none.
  - `word` holds the kind's payload inline when that payload is exactly 4 bytes, for example an `Identifier`'s interned name. Otherwise it is the offset in the store's `dataWords` where the payload starts.
  - Index 0 of the node table is a reserved dummy, so node indices are never 0.
- **Per store:** `nodes []nodeHeader`, `lists []nodeList`, `listNodes []uint32` and `dataWords []uint32`, shrunk to exact size after parsing.
- **Payload refs:** a child is a 4-byte node index. A list is a 4-byte list index, with 0 for none.
- **`nodeList`:** `{pos int32, end int32, start uint32, len uint32}`, 16 bytes. Its elements are `listNodes[start:start+len]`.
- **Node handle:** 8 bytes, `{file uint32, node uint32}`. The file id indexes a table of stores, and factory and transformer stores take ids from the same table.
- **Binder data moves out of payloads into per-store side tables:**
  - symbols and flow nodes go in a sparse node map: one `{bits uint32, base uint32}` block per 32 nodes, plus dense values found by popcount. It is built from a dense table once binding is done;
  - locals, local symbol, next container, and end, return and fallthrough flow nodes go in hash maps keyed by node index.
  - Deviation: the binder's node flags stay in the header's `flags` word instead of a sparse `nodeFlags` map. The binder sets and clears bits of the same word the parser does (reachability, `ContainsThis`, export context, error propagation), and every `Flags()` read would otherwise consult both. On vscode 262K nodes carry binder flags, so a sparse map would cost about 6MB (598K blocks of 8 bytes plus 4-byte values); the comparison should add that.
  - Deviation: a symbol is a `*Symbol`, so the sparse symbol map's values are 8 bytes, not 4.
- **Checker links:**
  - symbol-of-node is a per-file `map[uint32]` (a slice of maps indexed by file id);
    - Deviation: the per-store maps are found through a map keyed by store id, with a one-entry cache, not a slice. Store ids are process-wide slots, not indices into a program's file table, so a slice indexed by slot grows with every store alive in the process; the compiler test suite, with many programs alive at once, ran out of memory that way. Values are `*Symbol`, 8 bytes.
  - other node links are maps keyed by the 8-byte handle.

  There are no global node ids and no dense paged node link stores.
- **`GetSourceFileOfNode`** is O(1): `handle.file` gives the store, and the store gives its file.

### Record sizes, vscode

Payloads below exclude the binder fields and the subtree facts cache, which both move to side tables. Counts are of parsed nodes at the end of a vscode run: 19.39M nodes in 26,098 stores, and 3.61M lists holding 6.23M elements.

| Kind | Count | Now (bytes) | Payload | In `dataWords` | Header + data |
|---|---|---|---|---|---|
| Identifier | 6.65M | 28 | 4 | inline | 24 |
| PropertyAccessExpression | 1.74M | 40 | 12 | 12 | 36 |
| CallExpression | 1.14M | 44 | 16 | 16 | 40 |
| StringLiteral | 0.87M | 32 | 12 | 12 | 36 |
| PropertyAssignment | 0.69M | 48 | 20 | 20 | 44 |
| ExpressionStatement | 0.57M | 28 | 4 | inline | 24 |
| TypeReference | 0.43M | 28 | 8 | 8 | 32 |
| Block | 0.38M | 44 | 5 | 8 | 32 |
| VariableDeclaration | 0.36M | 48 | 16 | 16 | 40 |
| VariableDeclarationList | 0.36M | 28 | 4 | inline | 24 |
| BinaryExpression | 0.34M | 48 | 20 | 20 | 44 |
| VariableStatement | 0.34M | 36 | 8 | 8 | 32 |
| Parameter | 0.32M | 52 | 24 | 24 | 48 |
| ThisKeyword | 0.32M | 24 | 0 | – | 24 |

Tokens grow from 20 to 24 bytes. 8.3M nodes have an inline payload.

### Handle

`Node` is `struct{file, node uint32}`. It is 8 bytes and comparable, and the zero value is the absent node.

The handle size matters well beyond the AST: checker maps keyed by node, `[]Node` slices, symbol declarations and flow caches all hold nodes. Padding the current one-pointer handle to 16 bytes, with no other change, raised vscode's Memory used from 3,167,904K to 3,274,391K (+106MB). So a `{*store, index}` handle is out.

The file id indexes a process-wide table of stores. Each entry holds the bases of the store's four tables and the store itself, so `Kind()` is three dependent loads:

1. the table page, from a fixed global directory;
2. the entry's `nodes` base;
3. the header.

What Go forces:

- **Ids are recycled, so stores are owned and released.** A long-running language server creates stores without end (reparsed files, lazily parsed JSDoc, emit and checker working stores), so ids must be reused. A handle is just two integers, so it can't keep its store alive. Every store therefore has an owner that releases it explicitly; see "Ownership" below.
- **The table holds ordinary pointers.** Each entry holds the store and the bases of its four tables as atomic Go pointers the collector can see, so a store can't be freed while its slot is live, and a handle never reads freed memory. The table is paged (1,024 entries a page), so the collector scans only pages in use.
- **The id carries a generation.** An id is a 22-bit slot plus a 7-bit generation, bumped each time the slot is reused. Every access compares the entry's id with the handle's (one load and one compare), so a handle into a released store panics ("use of a node or list whose node store was released") instead of reading the store now in its slot. Released slots wait in a FIFO of 1,024 before reuse, so a stale handle goes undetected only after about 128k releases of other stores.
- **The bases are published atomically.** Parsed stores are private until they are frozen, so they grow by plain `append`. Some unfrozen stores are read by other goroutines while they grow, such as a source file's token factory, whose tokens are cached under a lock and then read without one. So a store publishes each new base with an atomic store, and readers load the bases atomically, which is a plain load on amd64.

### Ownership

The one rule is that **a store's owner outlives the uses of its nodes**. There is no `KeepAlive` or pinning anywhere: a node is valid exactly as long as whatever owns its store has not released it. A factory owns nothing; it is only an appender into a store that something with a clear lifetime owns (`ast.NewNodeFactory(store, hooks)`).

| Owner | Store(s) | Released by |
|---|---|---|
| Parsed `SourceFile` (held by its program or parse cache) | its own store, its lazily parsed JSDoc stores, and its token cache's store | `SourceFile.Release()` |
| `EmitContext` from `printer.NewEmitContext()` (one per file's transform and emit, per declaration-diagnostics query, per language-service request) | one working store, which its factory appends to; originals stay in the file's store | `EmitContext.Release()`, when emit for the file is done |
| `Checker` | one store for its synthetic nodes, isolated entity names, and node builder results kept by the caller | `Checker.Release()` |
| The checker's `typeToString` builder | its own context's store, whose nodes are only printed | at the end of the outermost `typeToString`-style call |
| `change.Tracker` | its `EmitContext`'s store, which fixes and organize imports build their nodes in | `Tracker.Release()`, after its changes are taken |
| Language-service request code that builds nodes (exhaustive case snippets, selection ranges, `printNode` and the API decoder) | a local `NodeStore` | `defer store.Release()` |
| Tree walks whose visitor may update nodes it skips (`astnav`, formatting, selection ranges) | a local, lazily created `NodeStore`; the updated nodes are never used | `defer store.Release()` |
| Program | the stores of synthetic import declarations | never (see below) |
| Tests | `parsetestutil.NewNodeStore(t)` and `NewNodeFactory(t)` | `t.Cleanup` |

A store refers into other stores freely (originals, reused children, parents of reused nodes) through its far table. It does not keep them alive: whoever uses a working store's nodes must also hold the owners of the stores they refer into, which in practice holds already (emit and the checker hold the program).

Not released yet, so they leak rather than break:

- **The CLI's program and checkers**: they live until the process exits.
- **Language-service source files and checkers.** The parse cache drops a file when its refcount reaches zero, but `RefCountCache.Ref` can resurrect a dropped entry and in-flight requests may still hold the old program, so releasing on drop could panic a request. Each reparse of an edited file leaks its store until a safe release point exists.
- **Program synthetic import stores**: a cloned program shares them and programs have no close. They are small.
- **Watch and build mode programs**, for the same reason as the CLI.

Misuse is caught, not silently wrong: making a node with a factory that has no store panics ("node factory has no store"), and reading a node of a released store panics as above. A missed release leaks memory, never corrupts it.

The serialized-type cache of the node builder may hold nodes of a released `typeToString` store; a lookup that finds one recomputes the entry (`Node.IsLive`), which loses some cache hits compared with before but changes no output.

### Cross-store refs and lists

Factory and transformer trees refer to nodes and lists of parsed files: originals, reused children, and parents of reused nodes. A plain 4-byte index can't name another store. This is a forced deviation: a ref with its top bit set is an index into the store's `far []Node` or `farLists []NodeList` table. Parsed files never refer outside their own store, so their refs are plain indices.

`NodeList` is a handle like `Node`, `{file, list}`. A `ModifierList`'s flags are computed from its elements, since the target `nodeList` has no room for them. Missing lists, which are rare, are kept in a per-store set.

### Other side tables

- **Subtree facts.** Today these are a lazily computed, atomically cached word in the payload of 76 composite kinds. They move to a per-store `[]atomic.Uint32` indexed by node, allocated on the first facts query. This keeps payloads at the target size. The checker never queries facts; emit, JSX module detection and some JS diagnostics do.
- **Go values.** Strings, `[]string`, `*SourceFile` and similar stay in the existing typed side tables, as 4-byte refs in payloads (text refs are 8 bytes), since pointer-free payloads can't hold them.
- **Flow nodes.** Flow nodes stay in a per-store table of 16-byte records. Their node field becomes a node ref.

### What gets deleted

Chunks, chunk headers and chunk codes, size-class and alignment logic, anchors, near/far self-relative refs, parent deltas, per-chunk id tables, `NodeIdGenerator` and the global node id counters, paged node link stores, list header encodings (derived pos and end, spilled lists), and the field offset tables' reliance on record headers.

`GetNodeId` survives as a key derived from the handle, `file<<32 | node`. It is unique among live nodes, which is all its remaining callers need: printer name caches, the declaration transformer and checker cache keys.

### Estimate (vscode)

| | Now | Target |
|---|---|---|
| Node records | 658MB | headers 465MB + `dataWords` 133MB = 598MB |
| Lists | ~54MB (8-byte headers plus elements) | 83MB (16 bytes each plus elements) |
| Binder data | ~73MB in records + 21MB symbol side table | sparse symbols ~26MB, sparse flows ~45MB, maps ~27MB (1.13M entries) |
| Far tables, id tables | ~16MB + ~35MB | far tables for factory stores only |
| Node link stores | paged by lazy id | maps keyed by handle (to be measured) |

The AST itself comes out roughly even, perhaps 50–80MB smaller. The link stores are the unknown: a map entry costs 2–3 times a dense page slot.

### Steps

Each step is a commit, measured separately:

1. Node links in maps keyed by node, and symbol-of-node as a node→symbol map, still on the current AST. Ids stay only for their few non-link users.
2. Nodes as (store, index): the tables, the 8-byte handle, the store table, payloads in `dataWords`, lists, and far tables. Chunks, anchors and ids are deleted. Binder fields and facts stay in payloads for now.
3. Binder data in sparse maps and hash maps.
4. Subtree facts in a per-store side table.
5. Symbol-of-node as per-file `map[uint32]`.

### Progress

Memory used from `--extendedDiagnostics`, single-threaded check. Base is `1fce2af19c3`; main is 3,792,305K on vscode.

| Step | vscode | sentry | mui |
|---|---|---|---|
| base | 3,168,196K | 1,829,589K | 1,174,464K |
| 1. node links in maps | 3,209,104K (+41MB) | 1,835,984K (+6MB) | 1,176,794K (+2MB) |
| 2. nodes as (store, index) | 3,281,105K (+72MB) | 1,851,797K (+16MB) | 1,187,501K (+11MB) |
| 3. binder side tables | 3,237,589K (−44MB) | 1,830,127K (−22MB) | 1,171,878K (−16MB) |
| 4. subtree facts table | 3,202,245K (−35MB) | 1,816,695K (−13MB) | 1,168,769K (−3MB) |
| 5. symbol-of-node per store | 3,188,922K (−13MB) | 1,818,221K (+2MB) | 1,170,622K (+2MB) |

Step 2's vscode heap: node headers 472MB, `dataWords` and `listNodes` 286MB, lists 54MB, in place of 742MB of chunks. The payloads still hold the binder fields and the facts word, which steps 3 and 4 move out; flow nodes grew from 47MB to 71MB because their table is never trimmed after binding.

Step 3 on vscode: `dataWords` and `listNodes` fall from 286MB to 195MB and flow nodes from 71MB to 44MB, now trimmed after binding. The sparse maps hold 2.62M symbols and 10.1M flow nodes (48MB of flow values and blocks plus 26MB of symbol values and blocks); the hash maps hold 557K next containers, 312K locals, 207K end flow nodes, 57K local symbols and 11K return flow nodes.

Step 4 removes the facts word from the payloads of the 76 composite kinds. The table that replaces it is made only for stores whose facts are queried, which a `--noEmit` check does for few files (JSX files, for module detection, which likely is why mui gains least); an emitting build makes it for every file it emits, at 4 bytes per node.

The generation check costs about 1s of vscode's ~22s check time (n=5 alternating, 23.1s with it, 22.1s without; the spread of each is about ±1s).

Step 5 keys symbol-of-node by store id, then node index. A slice indexed by slot would be cheaper to look up, but slots are process-wide and a test process makes hundreds of thousands of stores, so a per-checker slice grows with every store ever made; a map of maps grows only with the stores the checker touches.

### Final state

Sizes match the target: the header is 24 bytes, `nodeList` 16, and `Node` and `NodeList` handles 8. Payloads are as in the table under "Record sizes", with every kind's payload unchanged from it.

Memory used, final vs base: vscode +20.7MB (+0.65%), sentry −11.4MB (−0.6%), mui −3.8MB (−0.3%). Vs main, vscode is −603MB (−15.9%).

Check time, n=5 alternating, `taskset -c 2`, base vs final (min–max):

| | base | final | delta |
|---|---|---|---|
| vscode | 17.56–17.89s | 21.24–21.60s | +3.7s (+21%) |
| sentry | 14.37–14.67s | 16.13–16.52s | +1.7s (+12%) |
| mui | 5.62–6.39s | 6.13–6.72s | +0.5s (+9%) |

The CPU profile on vscode puts about 7% of samples in `storeEntryOf` and `slotEntry`: every header read loads the table page, then the entry, then the header, and checks the generation. The collector's share fell (mark workers 27% of samples after step 1, 18% at the end), since the tables are pointer-free and there are no chunk headers to scan, but not by enough to pay for the indirection.

Final vscode heap (in-use space at exit):

| | MB |
|---|---|
| Node headers (`nodes`) | 495 |
| `dataWords` and `listNodes` | 163 |
| `lists` | 59 |
| Binder sparse maps and hash maps | 133 |
| Flow nodes | 45 |
| Node links keyed by handle (`LinkStore`) | 135 |
| Symbol-of-node maps | 132 |
| Symbol links (`PagedLinkStore`, by symbol id) | 196 |

## Views and refs, no global table

Branch `new-ast-view`, from `new-ast-index` at `5e349ab704f`. The layout above stays: 24-byte headers, `dataWords`, 16-byte `nodeList`s, sparse binder side tables and lazy facts. What changes is how a node names its store. An 8-byte `{store id, index}` handle has to find its store through the process-wide store table on every access: the table page, the entry, then the header, plus the generation check. That is about 7% of CPU samples, and the check alone is about 1s on vscode. The table also forces explicit ownership: a handle can't keep its store alive, so every store needs an owner that releases it, owners that never release leak, and the language service has no safe point to release at.

The fix is to split the two jobs a handle does. Code that works on nodes holds a *view*, which carries a real pointer to its store. Code that keeps many nodes for a long time holds a *ref*, which is 8 bytes and is resolved through a table that its owner holds.

### The store is a plain struct

`NodeStore` (the old `nodeStore`, now exported) owns one file's tables, or a factory's or a transform's: `nodes`, `dataWords`, `lists`, `listNodes`, the side tables, the far tables, the binder data and the facts cache. It has no id, no slot, no generation and no `Release`. It lives as long as anything references it, like any Go object.

A store still publishes the bases of its four tables through atomic pointers, since a few stores are read by other goroutines while they grow (a source file's token cache). On amd64 an atomic load is a plain load.

### Views

```go
type Node struct {
    s    *NodeStore
    node uint32 // index in s.nodes, never 0 for a node
}                // 16 bytes, 4 of them padding
type NodeList struct {
    s    *NodeStore
    list uint32
}
```

`n.Kind()` is the store pointer's `nodes` base plus the index: two dependent loads, no table and no check. Methods keep their shape (`n.Kind()`, `n.Parent()`, `n.AsX().Field()`), so call sites don't change. Children, parents and list elements are views on the same store. The zero view is the absent node; reading a header through it panics, as on `new-ast`. The absent list still reads as empty.

Views are canonical: the padding is never used, so two views of a node are equal and views stay usable as map keys.

A field that refers to a node of another store (a factory node pointing at a parsed node, or a parsed node reused under a transformed parent) holds an index into its store's `far []Node`, as before. Those are real views, so the collector sees them and the referenced store stays alive as long as the referring one.

### Lifetime is reachability

Factories and transformers append into a store they are given. The emit context holds one working store per file's transform and emit, the checker one for its synthetic nodes, and language-service code makes stores where it needs them. A view of a node in any of them carries the store, so nothing else is needed: the store is collected when the last view, far slot or owner pointer goes away. `SourceFile.Release`, `EmitContext.Release`, `Checker.Release`, `NodeStore.Release`, `Node.IsLive`, the program's synthetic import stores and every `defer store.Release()` go away.

`GetSourceFileOfNode(n)` is `n.s.file`, or a walk up the parents for factory nodes, as before.

`GetNodeId` needs a number unique among live nodes, for names that embed it (pattern ambient modules, generated names), the checker's cache keys and the API's file handles. A store takes a 32-bit serial from a counter when it is made, and the id is `serial<<32 | index`. Nothing maps a serial back to a store.

### Refs

```go
type NodeRef struct {
    file uint32 // index in a FileTable
    node uint32
} // 8 bytes
```

A `FileTable` numbers stores for one program and is shared by the program's checkers. Indices `0..len(files)` are the program's files, in the program's order; the checkers' synthetic stores and any other store a checker meets (lazily parsed JSDoc, emit or language-service nodes passed to the checker) are numbered after them, the first time they are met. A ref is resolved with the table: a file's store is one slice index, `t.files[r.file]`; later stores are in fixed-size pages, so that readers need no lock while another checker adds one.

Going from a view to a ref needs the store's number in the table. A store can be in several programs at once (the language service shares parsed files between projects), at different positions, so the number can't be a field of the store alone. Each store instead caches the slot it has in the table that numbered it most recently: one atomic pointer to `{table, index}`. Converting a view is a load of that pointer and a compare with the table. A miss (a store first met, or one claimed since by another program) takes a lock, looks the store up or numbers it, and claims the cache. In a CLI build every store is in one program, so misses happen once per store.

There is no package-level registry: refs are meaningless without the table, and only the checker, which holds the program, resolves them. Code outside the checker (printer, transformers, language service, tests) uses views only.

### Which holder gets which

The checker keeps the most nodes for longest, so its bulk structures take refs:

- node links (`nodeLinks`, `signatureLinks`, `typeNodeLinks` and the other link stores keyed by node): maps keyed by `NodeRef`;
- symbol-of-node: a slice of per-file `map[uint32]*Symbol` indexed by `ref.file`, which the target layout asked for and the process-wide store ids ruled out;
- maps keyed by node (`moduleSymbols`, `contextFreeTypes`, `flowTypeCache`, `cachedArgumentsReferenced` and the like) and cache keys that hold node ids;
- symbol declarations and `valueDeclaration`, and `[]Node` held by types and symbols, where measurement shows it pays. The binder makes a file's symbols when the file is parsed, before any program numbers it, so binder symbols can't hold program refs; they keep views unless a cheaper per-file encoding pays for itself.

Everything else keeps views: the AST's own tables, the binder's side tables (which are indexed by node within their store), the printer, transformers and language service, and short-lived checker state.

The 8-byte handle was 16 bytes once before, padded everywhere, which cost +106MB on vscode. Each holder above is measured with views and with refs, and the choice recorded here.

### What gets deleted

The store table (`storePages`, `storeEntry`, `slotEntry`, `storeEntryOf`), store ids and their free list, generations and `checkStoreIDs`, `LiveNodeStores`, `StoreSlot`/`StoreID`, `IsLive`, every `Release`, the ownership table above and the symbol-of-node map keyed by store id.

### Steps

1. Views replace the store-table handle. The table, ids, generations and releases are deleted; checker structures keep views.
2. The checker's bulk structures hold `NodeRef`s through a per-program `FileTable`.

## Historical: earlier measurements (step 9 vs main, vscode, n=10, interleaved)

### Branch tip vs main

```
       │    main    │               arena               │
       │ check-sec  │ check-sec   vs base               │
VSCode   7.272 ± 2%   7.391 ± 2%  +1.64% (p=0.035 n=10)

       │    main    │            arena             │
       │ total-sec  │ total-sec   vs base          │
VSCode   8.249 ± 3%   8.360 ± 1%  ~ (p=0.197 n=10)

       │     main     │                arena                │
       │    mem-B     │    mem-B      vs base               │
VSCode   3.302Gi ± 0%   3.398Gi ± 0%  +2.92% (p=0.000 n=10)
```

`mem-B` is `MemStats.Alloc` after two forced GCs at the end of the run, i.e. the live heap.

### What did improve

- **GC mark CPU: 9.07s → 4.23s (−53%)**. Total CPU samples went from 48.95s to 44.11s (−10%).
- **Parse: about 5–10% faster.**
- **CPU-constrained (`taskset` to 4 cores): about 1–2% faster overall**, because GC can no longer hide on idle cores.
- 6 concurrent compiles on one machine: parity.

### Why wall-clock check time doesn't move

- On an unconstrained machine, Go's GC marks on otherwise idle cores. Halving mark work frees CPU that the checkers weren't going to use.
- Following AST references is only about **2–3% of check CPU**, so making it cheaper (or more expensive) barely shows up.
- About **15% of check CPU is map lookups**, mostly keyed by `*Symbol` or `string`:
  - `mergedSymbols map[*Symbol]*Symbol` (`getMergedSymbol`)
  - map-based symbol `LinkStore`s (checker.go ~695–723; node links already use a paged, ID-indexed store)
  - `getPropertyOfTypeEx` and symbol table lookups
  - `GetIdentifierToken`
  - `compareNodes` → `fileIndexMap`

  That is a checker data-structure problem, not an AST layout problem.

## Where the memory actually goes

Live heap at the end of a vscode run, from a heap profile taken right after the final forced GC. Each sample is attributed to the allocation site closest to the leaf:

| Category | main | arena |
|---|---|---|
| **AST** (records, lists, node slices) | 981MB | 897MB |
| AST side tables, text arena, stores | – | ~125MB |
| **AST total** | **~0.98GB** | **~1.02GB** |
| Source text | 110MB | 111MB |
| Binder symbols | 234MB | 248MB |
| Binder flow nodes | 71MB | 63MB |
| **Checker total** | **~2.0GB** | **~2.0GB** |
| …node links (paged) | 479MB | 472MB |
| …instantiated symbol tables / inherited members | ~650MB | ~650MB |
| …types | ~0.3–0.4GB | ~0.3–0.4GB |
| …checker symbols, symbol links, relations | ~200MB | ~200MB |
| **Total live** | 3.47GB | 3.56GB |

Peak RSS is 4.19GB on main and 4.21GB on the arena. Symbol (7.09M), type (2.43M) and identifier (4.52M) counts are identical.

### Why a "more compact" AST doesn't save memory

1. **Main is already compact.** Main's `Node` is also a 32-byte header (`Kind, Flags, Loc, id, Parent`, with no `data` interface), embedded at the start of each concrete node, and nodes come from per-kind slab pools rather than individual allocations. Arena records are only 0–16 bytes smaller per kind:

   | Kind | main | arena |
   |---|---|---|
   | Identifier | 56 | 48 |
   | StringLiteral | 56 | 48 |
   | PropertyAccessExpression | 72 | 64 |
   | CallExpression | 80 | 64 |
   | BinaryExpression | 88 | 80 |
   | TypeReference | 48 | 48 |
   | Token | 32 | 32 |

   The savings come from `string` (16 bytes) → `textRef` (8 bytes) and from pointer fields → 4-byte `sideRef`s. Together that's roughly 85MB.
2. **Side tables give it back, and more.** Moving `*Symbol` and `*FlowNode` out of the node replaces one 8-byte pointer with a 4-byte index plus an 8-byte table entry. These fields are populated on most declarations and identifiers, so this is a net loss, and the tables also carry append-growth slack and a clone on compaction. That costs about 125MB.
3. **Refs were 8 bytes.** These measurements predate step 10. With 32-bit refs, records round up to whole words, so many kinds don't shrink (Identifier: 32 + 4 + 8 → 48 bytes either way).
4. **The AST is only about 30% of the heap.** About 60% is checker state: instantiated symbol tables, node links, types and relations. This work doesn't touch any of that.

## Future work and expected impact

| Idea | What it is | Expected impact | Effort |
|---|---|---|---|
| **Inline symbol/flow IDs instead of side tables** | Give symbols and flow nodes dense uint32 IDs into global (or per-program) arrays and store the ID inline in the record, dropping the per-store side tables for densely populated fields. | About −125MB, recovering the side-table overhead. Lookup cost is similar (one indexed load). | Medium: needs stable ID allocation for binder- and checker-created symbols. |
| **24-byte header (drop `id`)** (done, step 28) | Move the 8-byte atomic id out of the header into per-chunk id tables. | −140MB on vscode (chunks −174MB, id tables +35MB). Per-chunk tables per generator could cut the 35MB further. | Medium: ids must stay unique and as dense as before for the paged node links. |
| **Lists in the arena** (done, step 15) | Store `NodeList`, `ModifierList` and `[]Node` as inline ref arrays in chunks instead of side-table slices. | About −50–100MB, fewer GC-visible slices, and better locality in `ForEachChild`. | Medium: the list API currently hands out `*NodeList` / `[]Node`, so it needs a view type. |
| **Compacted per-file image** | After parse and bind, copy a file's chunks into one exact-size block (with refs rewritten), dropping growth tails and side-table slack. | About −20–50MB. More importantly, a file becomes **one contiguous, pointer-free blob**: zero-copy hand-off to the JS API or IPC, mmap-able caches, and cheap incremental reuse. | Medium-high: cross-store refs and `keep` sets need fixing up, and this must happen before the file is shared. |
| **Relative refs again, done right** | 32-bit refs relative to the chunk or image base (not the field), with the image making "far" refs rare. | Shrinks every ref field by 4 bytes (about −100MB+). The previous attempt cost 6–8% of check time; with a contiguous image it should be one add instead of a branch plus a table lookup. | Medium, and depends on the image. |
| **Cheaper instantiated symbol tables** | `instantiateSymbolTable`, `addInheritedMembers` and `getNamedMembers` build about 650MB of maps and symbols. Use sorted slices or shared/lazy tables for small or derived members. | The largest single memory target, bigger than the whole AST saving potential. Likely a check-time win too (fewer map writes). | Medium-high; separate project. |
| **Dense symbol IDs in the checker** | Replace `map[*Symbol]…` link stores and `mergedSymbols` with ID-indexed paged arrays, as node links already are. | The actual check-time lever: up to about 10% of check CPU is in these maps. Independent of the AST work, but it compounds with it (and shares the symbol IDs from the first row). | Medium-high; separate project. |
| **Interned names** | Intern identifier text to dense IDs at parse time, and key symbol tables and `GetIdentifierToken` by ID instead of string. | Cuts string hashing in symbol table lookups: another few % of check CPU. Pairs naturally with `textRef`. | High: touches symbol tables broadly. |

### Rough combined picture

- **Inline IDs + header + lists + image:** the AST goes from about 1.02GB to roughly 0.7–0.8GB. Total memory goes from +3% to roughly −5–8% vs main, with GC mark CPU still about half. Check time is about neutral.
- **The bigger memory lever is the checker:** instantiated symbol tables and node links alone are about 1.1GB, more than the whole AST.
- **Plus dense symbol IDs and interned names:** this is where a real check-time win (high single digits to ~10%) would come from. The arena makes the AST cheap enough that the checker's maps are clearly the next bottleneck.
- **The image is the most interesting payoff:** a parsed and bound file as a single relocatable, pointer-free blob enables zero-copy API transfer and on-disk AST caching. A pointer-graph AST can't do that.
