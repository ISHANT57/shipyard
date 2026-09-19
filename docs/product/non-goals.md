# Non-goals

These are deliberately out of scope. Listing them prevents scope creep later
and gives future-me a reason to say no.

- **Not a hosted service.** No multi-tenant SaaS, no public deployment. Runs
  locally or in the operator's own environment only.
- **Not a CI/CD replacement.** Shipyard integrates with GitHub (via webhooks
  or Actions); it does not aim to replace GitHub Actions, GitLab CI, or
  Jenkins.
- **Not a Kubernetes platform.** `kind` may be used later purely to learn
  Kubernetes concepts (Phase 13); Shipyard does not target production K8s
  deployment as a goal.
- **No paid infrastructure.** No paid cloud, no paid database, no paid
  monitoring, no paid LLM API as a runtime dependency. Free/local/open-source
  only.
- **No message broker.** No Kafka, no RabbitMQ. PostgreSQL is the queue —
  chosen deliberately, see the queue ADR, not because a broker was
  unavailable.
- **No microservices for their own sake.** Two planes (control, execution),
  not ten services, unless a real problem forces the split.
- **No claim of exactly-once processing** unless a specific operation has a
  precise, defensible, tested definition of that guarantee.
- **No security guarantee beyond documented scope.** Docker-based isolation
  is defense-in-depth, not an absolute boundary against hostile code — this
  is stated explicitly wherever the sandbox is described.
