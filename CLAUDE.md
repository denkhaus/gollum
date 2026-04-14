# Gollum Project - Infos for the coding agent

Use the `karpathy-guidelines` skill. It is our mental model and key principles of our work, strictly to follow.

# What's next

To know what's next to do in this repo use te `forgejo-cli` skill.
Then use `forgejo issue ready --repo denkhaus/gollum` to lern about the next ready tasks.

# Knowledge Base

- `/home/denkhaus/dev/kb`

## Guidelines you must strictly adhere

- [Go Programming Guide](/home/denkhaus/dev/kb/guides/guide.golang.programming.md)
- [Go Dependency Injection Guide](/home/denkhaus/dev/kb/guides/guide.golang.di.md)
- [Go Testing Guide](/home/denkhaus/dev/kb/guides/guide.golang.testing.md)
- [Go Config Guide](/home/denkhaus/dev/kb/guides/guide.golang.config.md)
- [Go Logging Guide](/home/denkhaus/dev/kb/guides/guide.golang.logging.md)
- [Langfuse Tracing Guide](/home/denkhaus/dev/kb/guides/guide.golang.langfuse-tracing.md)

## General rules

- Explore lifecycle functions running `just`
- Whenever you build the app for testing reasons run `just build`
- General helper functions MUST be implemented in the `shared` package to be reusable from other packages
