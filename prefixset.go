package netipds

import (
	"fmt"
	"net/netip"
)

// PrefixSetBuilder builds an immutable [PrefixSet].
//
// The zero value is a valid PrefixSetBuilder representing a builder with zero
// Prefixes.
//
// Call PrefixSet to obtain an immutable PrefixSet from a PrefixSetBuilder.
//
// If Lazy == true, then path compression is delayed until a PrefixSet is
// created. The builder itself remains uncompressed. Lazy mode can dramatically
// improve performance when building large PrefixSets.
type PrefixSetBuilder struct {
	Lazy bool
	tree tree[bool]
}

// Add adds p to s.
func (s *PrefixSetBuilder) Add(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	if s.Lazy {
		s.tree = *(s.tree.insertLazy(keyFromPrefix(p), true))
	} else {
		s.tree = *(s.tree.insert(keyFromPrefix(p), true))
	}
	return nil
}

// Remove removes p from s. Only the exact Prefix provided is removed;
// descendants are not.
//
// To remove entire sections of IP space at once, see
// [PrefixSetBuilder.Filter], [PrefixSetBuilder.Subtract] and
// [PrefixSetBuilder.SubtractPrefix].
func (s *PrefixSetBuilder) Remove(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	s.tree.remove(keyFromPrefix(p))
	return nil
}

// Filter removes all Prefixes that are not encompassed by o from s.
func (s *PrefixSetBuilder) Filter(o *PrefixSet) {
	s.tree.filter(&o.tree)
}

// SubtractPrefix modifies s so that p and all of its descendants are removed,
// leaving behind any remaining portions of affected Prefixes. This may add
// elements to fill in gaps around the subtracted Prefix.
//
// For example, if s is {::0/126}, and we subtract ::0/128, then s will become
// {::1/128, ::2/127}.
func (s *PrefixSetBuilder) SubtractPrefix(p netip.Prefix) error {
	if !p.IsValid() {
		return fmt.Errorf("Prefix is not valid: %v", p)
	}
	s.tree.subtractKey(keyFromPrefix(p))
	return nil
}

// Subtract modifies s so that the Prefixes in o, and all of their
// descendants, are removed from s, leaving behind any remaining portions of
// affected Prefixes. This may add elements to fill in gaps around the
// subtracted Prefixes.
//
// For example, if s is {::0/126}, and we subtract ::0/128, then s will become
// {::1/128, ::2/127}.
func (s *PrefixSetBuilder) Subtract(o *PrefixSet) {
	s.tree = *s.tree.subtractTree(&o.tree)
}

// Intersect modifies s so that it contains the intersection of the entries
// in s and o: to be included in the result, a Prefix must either (a) exist in
// both sets or (b) exist in one set and have an ancestor in the other.
func (s *PrefixSetBuilder) Intersect(o *PrefixSet) {
	s.tree = *s.tree.intersectTree(&o.tree)
}

// Merge modifies s so that it contains the union of the entries in s and o.
func (s *PrefixSetBuilder) Merge(o *PrefixSet) {
	s.tree = *s.tree.mergeTree(&o.tree)
}

// PrefixSet returns an immutable PrefixSet representing the current state of s.
//
// The builder remains usable after calling PrefixSet.
func (s *PrefixSetBuilder) PrefixSet() *PrefixSet {
	t := s.tree.copy()
	if s.Lazy && t != nil {
		t = t.compress()
	}
	return &PrefixSet{*t, t.size()}
}

// String returns a human-readable representation of s's tree structure.
func (s *PrefixSetBuilder) String() string {
	return s.tree.stringImpl("", "", true)
}

// PrefixSet is a set of [netip.Prefix] values. It is implemented as a binary
// radix tree.
//
// PrefixSet offers unique functionality beyond what a PrefixMap[bool] can do.
// In particular, during the building stage (PrefixSetBuilder) you can combine
// sets in useful ways using methods like [PrefixSetBuilder.Merge],
// [PrefixSetBuilder.Intersect], and [PrefixSetBuilder.Subtract].
//
// Use [PrefixSetBuilder] to construct PrefixSets.
type PrefixSet struct {
	tree tree[bool]
	size int
}

// Contains returns true if this set includes the exact Prefix provided.
func (s *PrefixSet) Contains(p netip.Prefix) bool {
	return s.tree.contains(keyFromPrefix(p))
}

// Encompasses returns true if this set includes a Prefix which completely
// encompasses p. The encompassing Prefix may be p itself.
func (s *PrefixSet) Encompasses(p netip.Prefix) bool {
	return s.tree.encompasses(keyFromPrefix(p), false)
}

// EncompassesStrict returns true if this set includes a Prefix which
// completely encompasses p. The encompassing Prefix must be an ancestor of p,
// not p itself.
func (s *PrefixSet) EncompassesStrict(p netip.Prefix) bool {
	return s.tree.encompasses(keyFromPrefix(p), true)
}

// OverlapsPrefix returns true if this set includes a Prefix which overlaps p.
func (s *PrefixSet) OverlapsPrefix(p netip.Prefix) bool {
	return s.tree.overlapsKey(keyFromPrefix(p))
}

func (s *PrefixSet) rootOf(
	p netip.Prefix,
	strict bool,
) (outPfx netip.Prefix, ok bool) {
	label, _, ok := s.tree.rootOf(keyFromPrefix(p), strict)
	if !ok {
		return outPfx, false
	}
	return label.toPrefix(), true
}

// RootOf returns the shortest-prefix ancestor of p in s, if any.
// If p itself has an entry and has no ancestors, then p's entry is returned.
func (s *PrefixSet) RootOf(p netip.Prefix) (netip.Prefix, bool) {
	return s.rootOf(p, false)
}

// RootOfStrict returns the shortest-prefix ancestor of p in s, if any. If p
// has no ancestors in s, then RootOfStrict returns zero values and false.
func (s *PrefixSet) RootOfStrict(p netip.Prefix) (netip.Prefix, bool) {
	return s.rootOf(p, true)
}

func (s *PrefixSet) parentOf(
	p netip.Prefix,
	strict bool,
) (outPfx netip.Prefix, ok bool) {
	key, _, ok := s.tree.parentOf(keyFromPrefix(p), strict)
	if !ok {
		return outPfx, false
	}
	return key.toPrefix(), true
}

// ParentOf returns the longest-prefix ancestor of p in s, if any. If p itself
// has an entry, then p's entry is returned.
func (s *PrefixSet) ParentOf(p netip.Prefix) (netip.Prefix, bool) {
	return s.parentOf(p, false)
}

// ParentOfStrict returns the longest-prefix ancestor of p in s, if any.
// If p has no ancestors in the set, then ParentOfStrict returns zero values
// and false.
func (s *PrefixSet) ParentOfStrict(p netip.Prefix) (netip.Prefix, bool) {
	return s.parentOf(p, true)
}

// DescendantsOf returns a PrefixSet containing all descendants of p in s,
// including p itself if it has an entry.
func (s *PrefixSet) DescendantsOf(p netip.Prefix) *PrefixSet {
	t := s.tree.descendantsOf(keyFromPrefix(p), false)
	return &PrefixSet{*t, t.size()}
}

// DescendantsOfStrict returns a PrefixSet containing all descendants of p in
// s, excluding p itself.
func (s *PrefixSet) DescendantsOfStrict(p netip.Prefix) *PrefixSet {
	t := s.tree.descendantsOf(keyFromPrefix(p), true)
	return &PrefixSet{*t, t.size()}
}

// AncestorsOf returns a PrefixSet containing all ancestors of p in s,
// including p itself if it has an entry.
func (s *PrefixSet) AncestorsOf(p netip.Prefix) *PrefixSet {
	t := s.tree.ancestorsOf(keyFromPrefix(p), false)
	return &PrefixSet{*t, t.size()}
}

// AncestorsOfStrict returns a PrefixSet containing all ancestors of p in s,
// excluding p itself.
func (s *PrefixSet) AncestorsOfStrict(p netip.Prefix) *PrefixSet {
	t := s.tree.ancestorsOf(keyFromPrefix(p), true)
	return &PrefixSet{*t, t.size()}
}

// Prefixes returns a slice of all Prefixes in s.
func (s *PrefixSet) Prefixes() []netip.Prefix {
	res := make([]netip.Prefix, s.size)
	i := 0
	s.tree.walk(key{}, func(n *tree[bool]) bool {
		if n.hasEntry {
			res[i] = n.key.toPrefix()
			i++
		}
		return i >= len(res)
	})
	return res
}

// PrefixesCompact returns a slice of the Prefixes in s that are not
// children of other Prefixes in s.
//
// Note: PrefixCompact does not merge siblings, so the result may contain
// complete sets of sibling prefixes, e.g. 1.2.3.0/32 and 1.2.3.1/32.
func (s *PrefixSet) PrefixesCompact() []netip.Prefix {
	res := make([]netip.Prefix, 0, s.size)
	s.tree.walk(key{}, func(n *tree[bool]) bool {
		if n.hasEntry {
			res = append(res, n.key.toPrefix())
			return true
		}
		return false
	})
	return res
}

// String returns a human-readable representation of the s's tree structure.
func (s *PrefixSet) String() string {
	return s.tree.stringImpl("", "", true)
}

// Size returns the number of elements in s.
func (s *PrefixSet) Size() int {
	return s.size
}
