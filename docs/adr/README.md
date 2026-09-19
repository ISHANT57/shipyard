# Architecture Decision Records

An ADR records one significant technical decision: the context that forced
it, the options considered, and why one was chosen over the others. It
exists so that six months from now — or six minutes from now, mid-review —
the reasoning is still there, not just the outcome.

## When to write one

Write an ADR when a decision is hard to reverse, affects more than one
component, or trades off real qualities against each other (e.g.
simplicity vs performance, consistency vs availability). Do not write one
for a decision with an obvious, low-cost answer — that just decorates the
folder.

## Process

1. Copy [TEMPLATE.md](TEMPLATE.md) to `ADR-NNN-short-title.md`, numbered
   sequentially.
2. Fill in Context and Options first, before Decision — the point is to
   show the reasoning was real, not reconstructed after the fact.
3. Status starts as `Proposed`. It becomes `Accepted` once the decision is
   final (usually when its PR merges), `Superseded by ADR-NNN` if a later
   decision replaces it, or `Rejected` if abandoned before acceptance.
4. Once `Accepted`, an ADR is not edited to reflect new information — a new
   ADR supersedes it instead. History stays visible.

## Index

Filled in as ADRs are written, starting in Phase 01 (ADR-001 onward — see
[phase-01-architecture.md](../phases/phase-01-architecture.md) for the
planned list).
