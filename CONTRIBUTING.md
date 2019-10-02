# Contributing to 0-OS

The following sections outline the process all changes to the repositories go through.  
All changes, regardless of whether they are from newcomers to the community or  
from the core team follow the same process and are given the same level of review.

- [Team values](#team-values)
- [Contributing a feature](#contributing-a-feature)
- [Setting up to contribute](#setting-up-a-development-environment)
- [Pull requests](#pull-requests)
- [Issues](#issues)

## Team values

We promote and encourage a set of [shared values](VALUES.md) to improve our
productivity and inter-personal interactions.

## Contributing a feature

In order to contribute a feature you'll need to go through the following steps:

- Create a Github issue to explain the feature you want to create. A member of the core team will engage with
you and together you will decide if the feature could be accepted and what is the best approach to take to implement it.

- Once the general idea has been accepted, create a draft PR that contains a design document. The design document should explain how the feature is going to be implemented, eventually a test plan and any technical information that could be interesting for other team member to know.

- Submit PRs with your code changes.

- Submit PRs with documentation for your feature, including usage examples when possible

> Note that we prefer bite-sized PRs instead of giant monster PRs. It's therefore preferable if you
can introduce large features in smaller reviewable changes that build on top of one another.

If you would like to skip the process of submitting an issue and
instead would prefer to just submit a pull request with your desired
code changes then that's fine. But keep in mind that there is no guarantee
of it being accepted and so it is usually best to get agreement on the
idea/design before time is spent coding it. However, sometimes seeing the
exact code change can help focus discussions, so the choice is up to you.

## Setting up a development environment

Check out the [documentation](https://github.com/threefoldtech/zos/tree/master/docs) learn about the code
base and setting up your [development environment](https://github.com/threefoldtech/zos/blob/master/qemu/README.md).

## Pull requests

If you're working on an existing issue, simply respond to the issue and express
interest in working on it. This helps other people know that the issue is
active, and hopefully prevents duplicated efforts.

To submit a proposed change:

- Fork the affected repository.

- Create a new branch for your changes.

- Develop the code/fix.

- Add new test cases. In the case of a bug fix, the tests should fail
  without your code changes. For new features try to cover as many
  variants as reasonably possible.

- Modify the documentation as necessary.

- Verify the entire CI process (building and testing) works.

While there may be exceptions, the general rule is that all PRs should
be 100% complete - meaning they should include all test cases and documentation
changes related to the change.

See [Writing Good Pull Requests](https://github.com/istio/istio/wiki/Writing-Good-Pull-Requests) for guidance on creating
pull requests.

## Issues

[GitHub issues](https://github.com/threefoldtech/zos/issues/new) can be used to report bugs or submit feature requests.

When reporting a bug please include the following key pieces of information:

- The version of the project you were using (e.g. version number,
  or git commit)

- Any logs that contains some error message or usefull information related to the bug

- The exact, minimal, steps needed to reproduce the issue.
  Submitting a 5 line script will get a much faster response from the team
  than one that's hundreds of lines long.
