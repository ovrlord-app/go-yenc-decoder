# Contributing to go-yenc-decoder

We want to make contributing to this project as easy and transparent as possible, whether it's:

Reporting a bug
Discussing the current state of the code
Submitting a fix
Proposing new features
Becoming a maintainer

## Table of Contents

1. [Code of Conduct](#code-of-conduct)
2. [We Develop with GitHub](#we-develop-with-github)
  - [Report bugs using GitHub's issues](#report-bugs-using-githubs-issues)
  - [We Use Github Flow](#we-use-github-flow)
3. [Coding Conventions](#coding-conventions)
  - [Use of AI Agents](#use-of-ai-agents) 
  - [Go code conventions](#go-code-conventions)
4. [Authors](#authors)
5. [License](#license)

## Code of Conduct

Please read the [Code of Conduct](CODE_OF_CONDUCT.md) document.

## We Develop with GitHub

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

### Report bugs using GitHub's [issues](https://github.com/ovrlord-app/go-yenc-decoder/issues)

We use GitHub issues to track public bugs. Report a bug by [opening a new issue](); it's that easy!

### We Use [Github Flow](https://docs.github.com/en/get-started/quickstart/github-flow)

Pull requests are the best way to propose changes to the codebase we
use [GitHub Flow](https://docs.github.com/en/get-started/quickstart/github-flow). We actively welcome your pull
requests:

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed/added APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Coding Conventions

### Use of AI Agents

AI agents and code generation tools may be used thoughtfully to produce quality contributions. However:
- Generated code must adhere to the established codebase standards and conventions outlined in this guide.
- Pull requests must not include unnecessary artifacts, verbose comments, or boilerplate that is typical byproducts of AI generation processes.
- Do not use AI to "vibe code" entire features without careful understanding and review of the generated implementation.
- All code, whether AI-assisted or manually written, must be reviewed carefully for correctness, maintainability, and alignment with the project's architectural principles.
- Contributors are responsible for ensuring the quality and appropriateness of any AI-generated code they submit.

### Go code conventions

- Follow [Effective Go](https://go.dev/doc/effective_go)
  and [Code Review Comments Guide](https://go.dev/wiki/CodeReviewComments) from the Go project as much as
  possible within reason.
- Go is not an Object-Oriented Programming language, we favor simplicity.
- Use standard library packages as much as possible, new dependencies should come with a valid reason for adding
  another dependency.
- All Go code is linted with `golangci-lint` on every commit.

## Authors

See the list of [contributors](https://github.com/ovrlord-app/go-yenc-decoder/contributors) who
participated in
this project.

## License

When you submit code changes, your submissions are understood to be under the
same [MIT License](../LICENSE) that covers the project. Feel free to contact the
maintainers if that's a concern.
