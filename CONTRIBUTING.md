# Contributing Guidelines

Parca is licensed under the [Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0) license and accept contributions via GitHub pull requests. This document outlines some of the conventions on development workflow, commit message formatting, contact points and other resources to make it easier to get your contribution accepted.

# Certificate of Origin

By contributing to this project you agree to [sign a Contributor License Agreement(CLA)](https://cla-assistant.io/parca-dev/parca).

# Code of Conduct

Parca follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). Please contact the Parca maintainers at parca-team@googlegroups.com to report any CoC violations.
# Prerequisites

Install the following dependencies (Instructions are linked for each dependency).

- [Go](https://golang.org/doc/install)
- [Node](https://nodejs.org/en/download/)
- [Docker](https://docs.docker.com/engine/install/)
- [minikube](https://v1-18.docs.kubernetes.io/docs/tasks/tools/install-minikube/)
- [kubectl](https://v1-18.docs.kubernetes.io/docs/tasks/tools/install-kubectl/)

# Getting Started

Fork and clone the [parca](https://github.com/parca-dev/parca) repository on GitHub to your machine.

```
$ git clone git@github.com:parca-dev/parca.git

```

Go to the project directory and compile parca:

```
$ cd parca

$ make build
```

Run the binary locally.

```
./bin/parca
```
Once compiled the server ui can be seen at http://localhost:7070.


To profile all containers using Kubernetes, the parca-server can be run alongside parca-ui using Tilt.

```
$ cd parca

$ make dev/up

$ tilt up
```

Test your changes by running:
```
$ cd parca && make go/test
```


<!--
TODO:
    # add Once you are done, you can close the kvm instances by using make dev/down

    #Internals
        ## Code Structure
-->

# Making a PR

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change. If you are not entirely sure about this, you can discuss this on the [Parca Discord](https://discord.gg/ZgUpYgpzXy) server as well.

Please make sure to update tests as appropriate.

This is roughly what the contribution workflow should look like:

- Create a topic branch from where you want to base your work (usually main).
- Make commits of logical units.
- Make sure the tests pass, and add any new tests as appropriate.
- Make sure your commit messages follow the commit guidelines (see below).
- Push your changes to a topic branch in your fork of the repository.
- Submit a pull request to the original repository.

Thank you for your contributions!


# Commit Guidelines

We follow a rough convention for commit messages that is designed to answer two
questions: what changed and why. The subject line should feature the what and
the body of the commit should describe the why.

```

scripts: add the test-cluster command

this uses tmux to setup a test cluster that you can easily kill and
start for debugging.

Fixes #38

```

The first line is the subject and should be no longer than 70 characters, the second line is always blank, and other lines should be wrapped at 80 characters. This allows the message to be easier to read on GitHub as well as in various git tools.


# UI Project - Code Formatting Guidelines

We use [Prettier](https://prettier.io/docs/en/options.html) for code formatting the files in the UI project. The following are the configuration overrides over Prettier's defaults:

1. `printWidth`: `100`
2. `singleQuote`: `true`
3. `bracketSpacing`: `false`
4. `arrowParens`: `'avoid'`
