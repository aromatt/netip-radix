# netipds
[![Go Reference](https://pkg.go.dev/badge/github.com/aromatt/netipds)](https://pkg.go.dev/github.com/aromatt/netipds)
[![Go Report Card](https://goreportcard.com/badge/github.com/aromatt/netipds)](https://goreportcard.com/report/github.com/aromatt/netipds)
[![codecov](https://codecov.io/gh/aromatt/netipds/graph/badge.svg?token=WJ1JHSM05F)](https://codecov.io/gh/aromatt/netipds)

This package builds on the
[netip](https://pkg.go.dev/net/netip)/[netipx](https://pkg.go.dev/go4.org/netipx)
family, adding two immutable, tree-based collection types for [netip.Prefix](https://pkg.go.dev/net/netip#Prefix):
* `PrefixMap[T]` - for associating data with IPs and prefixes and fetching that data with network hierarchy awareness
* `PrefixSet` - for storing sets of prefixes and combining those sets in useful ways (unions, intersections, etc)

Both are backed by a binary [radix tree](https://en.wikipedia.org/wiki/Radix_tree),
which enables a rich set of efficient queries about prefix containment, hierarchy,
and overlap.

### Goals
* *Efficiency* - this package aims to provide fast, immutable, thread-safe collection types for IP networks.
* *Integration with `net/netip`* - this package is built around `netip.Prefix` (to understand the benefits of this IP type, see this excellent [post](https://tailscale.com/blog/netaddr-new-ip-type-for-go) by Tailscale about the predecessor to `net/netip`).
* *Completeness* - most other radix tree IP libraries lack several of the queries provided by `netipds`.

### Non-Goals
* *Mutability* - for use cases requiring continuous mutability, try [kentik/patricia](https://github.com/kentik/patricia).
* *Persistence* - this package is for data sets that fit in memory.
* *Non-IP network keys* - the collections in this package support exactly one key type: `netip.Prefix`.

## Usage
Usage is similar to that of [netipx.IPSet](https://pkg.go.dev/go4.org/netipx#IPSet):
to construct a `PrefixMap` or `PrefixSet`, use the respective builder type.

### Example
```go
// Make our examples more readable
px := netip.MustParsePrefix

// Build a PrefixMap
pmb := PrefixMapBuilder[string]{}
pmb.Set(px("1.2.0.0/16"), "hello")
pmb.Set(px("1.2.3.0/24"), "world")
pm := pmb.PrefixMap()

// Fetch an exact entry from the PrefixMap.
val, ok := pm.Get(px("1.0.0.0/16"))              // => ("hello", true)

// Ask if the PrefixMap contains an exact
// entry.
ok = pm.Contains(px("1.2.3.4/32"))               // => false

// Ask if a Prefix has any ancestor in the
// PrefixMap.
ok = pm.Encompasses(px("1.2.3.4/32"))            // => true

// Fetch a Prefix's nearest ancestor.
p, val, ok := pm.ParentOf(px("1.2.3.4/32"))      // => (1.2.3.0/24, "world", true)

// Fetch all of a Prefix's ancestors, and
// convert the result to a map[Prefix]string.
m := pm.AncestorsOf(px("1.2.3.4/32")).ToMap()    // => map[1.2.0.0/16:"hello"
                                                 //        1.2.3.0/24:"world"]

// Fetch all of a Prefix's descendants, and
// convert the result to a map[Prefix]string.
m = pm.DescendantsOf(px("1.0.0.0/8")).ToMap()    // => map[1.2.0.0/16:"hello"
                                                 //        1.2.3.0/24:"world"]
```

### Set Operations with PrefixSet
`PrefixSet` offers set-specific functionality beyond what can be done with
`PrefixMap`.

In particular, during the building stage, you can combine sets in the following ways:

|Operation|Method|Result|
|---|---|---|
|**Union**|[PrefixSetBuilder.Merge](https://pkg.go.dev/github.com/aromatt/netipds#PrefixSetBuilder.Merge)|Every prefix found in either set.|
|**Intersection**|[PrefixSetBuilder.Intersect](https://pkg.go.dev/github.com/aromatt/netipds#PrefixSetBuilder.Intersect)|Every prefix that either (1) exists in both sets or (2) exists in one set and has an ancestor in the other.|
|**Difference**|[PrefixSetBuilder.Subtract](https://pkg.go.dev/github.com/aromatt/netipds#PrefixSetBuilder.Subtract)|The difference between the two sets. When a child is subtracted from a parent, the child itself is removed, and new elements are added to fill in remaining space.|

## Related packages

### https://github.com/kentik/patricia

This package uses a similar underlying data structure, but its goal is to provide
mutability while minimizing garbage collection cost. By contrast, netipds aims to
provide immutable (and thus GC-friendly) collection types that integrate well with
the netip family and offer a comprehensive API.
