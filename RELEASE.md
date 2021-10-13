# Release schedule

To be discussed.

# How to cut a new release

> This guide is strongly based on the [Prometheus release instructions](https://github.com/prometheus/prometheus/blob/main/RELEASE.md).

## Branch management and versioning strategy

We use [Semantic Versioning](http://semver.org/).

We maintain a separate branch for each minor release, named `release-<major>.<minor>`, e.g. `release-1.1`, `release-2.0`.

The usual flow is to merge new features and changes into the main branch and to merge bug fixes into the latest release branch. Bug fixes are then merged into main from the latest release branch. The main branch should always contain all commits from the latest release branch.

If a bug fix got accidentally merged into main, cherry-pick commits have to be created in the latest release branch, which then have to be merged back into main. Try to avoid that situation.

Maintaining the release branches for older minor releases happens on a best effort basis.

## Prepare your release

For a new major or minor release, work from the `main` branch. For a patch release, work in the branch of the minor release you want to patch (e.g. `release-0.3` if you're releasing `v0.3.2`).

## Publish the new release

For new minor and major releases, create the `release-<major>.<minor>` branch starting at the PR merge commit.

From now on, all work happens on the `release-<major>.<minor>` branch.

### Via GitHub's UI

Go to https://github.com/parca-dev/parca/releases/new and click on "Choose a tag" where you can type the new tag name.
Click on "Create new tag" in the dropdown and make sure `main` is selected for a new major or minor release or the `release-<major>.<minor>` branch for a patch release. 
The title of the release is the tag itself.  
You can generate the changelog and then add additional contents from previous a release (like social media links and more).

### Via CLI

Alternatively, you can do the tagging on the commandline:

Tag the new release with a tag named `v<major>.<minor>.<patch>`, e.g. `v2.1.3`. Note the `v` prefix.

```bash
git tag -s "v2.1.3" -m "v2.1.3"
git push origin "v2.1.3"
```

Signed tag with a GPG key is appreciated, but in case you can't add a GPG key to your Github account using the following [procedure](https://help.github.com/articles/generating-a-gpg-key/), you can replace the `-s` flag by `-a` flag of the `git tag` command to only annotate the tag without signing.

## Final steps

Our CI pipeline will automatically push the container images to [ghcr.io](ghcr.io/parca-dev/parca).

Go to https://github.com/parca-dev/parca/releases and check the created release.

For patch releases, submit a pull request to merge back the release branch into the `main` branch.

Take a breath. You're done releasing.
