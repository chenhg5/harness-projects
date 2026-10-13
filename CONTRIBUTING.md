# Contributing to Harness Projects

Thank you for helping define an open, portable project standard.

## Design principles

1. The project is the primary unit of work.
2. Repositories are project resources and may be numerous.
3. Harness adapters are views of the canonical project files.
4. Files should remain readable and useful without a specific tool.
5. New features should degrade gracefully in unsupported harnesses.

## Proposals

For changes to the standard, add or update the relevant document under
`spec/`. Include examples and describe migration behavior. Keep vendor-specific
behavior in `adapters/` unless it changes the portable project contract.

## Pull requests

Please keep pull requests focused. A useful pull request includes:

- the problem being solved;
- the proposed contract or behavior;
- compatibility and migration notes;
- examples or validation steps where applicable.
