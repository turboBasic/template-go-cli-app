# Technical debt

Known departures from the architecture specification and the ADRs, each with a condition the repository can
evaluate. An entry is removed in the change that makes its condition true.

## Open entries

### Array settings are promised but unsupported

Specification section 5 and [ADR-0007](decisions/0007-strict-configuration-and-path-resolution.md) say an array
is replaced when configuration layers merge. No setting is an array: the loader rejects array values and
`valueTypeOf` panics on a slice field, so only `TestMergeLayerRecursivelyAndReplacesArrays` exercises the promise.

Remove this entry when a setting with an array type exists and its replacement is tested through the loader, or
when an ADR superseding ADR-0007 narrows the promise.
