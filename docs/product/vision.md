# Vision

Shipyard is a local-first developer automation platform. A developer points it
at a repository; Shipyard detects the project, runs it through a pipeline of
validation, testing, security scanning and build stages inside isolated
containers, and returns a structured, evidence-backed answer to one question:

> **Is this project safe and ready to ship?**

It exists to teach the operator (and demonstrate to others) how a real
engineering-automation platform is built: durable job processing, container
isolation, supply-chain security, and observability — not to replace any
existing CI product.

## Who it is for

Right now: one developer (the author), building it to learn production-grade
backend and platform engineering. The design keeps a second, real audience in
mind — a team lead who receives a pull request and wants confidence it is
safe to merge, without reading every diff or scan report by hand.

## What "done" looks like

A stranger can clone the repo, run one command, point it at a sample project,
and get back a report explaining what passed, what failed, and why — with
traces and metrics showing exactly where time was spent and what broke when
something was deliberately broken.
