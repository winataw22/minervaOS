# Releases of Zero-OS

We use a simple pipeline release workflow. Building and file distribution are made using GitHub Actions.
Usable files are available on the [Zero-OS Hub](https://hub.grid.tf/tf-zos).

Under this hub repo you can find 4 different `tag links` (ie. links to tags):

- development
- qa
- testing
- production

A `tag` in the hub terminology is when multiple flist gets some `tag` then those flists appear grouped together under one directory which is the `tag` name. This means multiple flists that are built together or belong to a certain unique entity can be grouped for an ease of management and processing.

A `tag link` is a link that exists in a repo to a tag in another repo. We then use this feature to link from [tf-zos](https://hub.grid.tf/tf-zos) repo to a `build` tag under [tf-autobuilder](https://hub.grid.tf/tf-autobuilder/) repo.

For example a `development` tag link from tf-zos will point to a tag (say `61cc487`) under tf-autobuilder. What does that mean? it means the development env zos is using this build tag, and the flists installed (and used by the zos nodes) in development are installed from that build tag.

Before createing a new release we need to make sure that the chain(substrate client) is up to date.
```sh
cd pkg
go get 'github.com/threefoldtech/tfchain/clients/tfchain-client-go@<latest version>'
```

On creating a new release, the build tag will get that exact release version (say v3.20.0), instead of a commit short hash.

For more details on how the system updates itself please check [upgrade documentation](../internals/identity/upgrade.md).

## Building

On a push to main branch on the zos repository, a new development build is triggered.  This builds ALL zos packages (main zos flist) and also all the [runtime packages](../../bins/packages/). All packages are tagged with the `short commit` hash. This means all built packages will appear under the `tf-autobuilder/<hash>` tag.

Once the building process is over, the `tag link` **development** under `tf-zos` is then updated to point to the latest build tag.

## Releases

On creating a release it's exactly the same as above except the tag will be the `release` version. This means that releasing a certain version to a specific network is as easy as creating the proper `tag link` from `tf-zos` to the corresponding tag under `tf-autobuilder` for example:

```bash
production -> ../tf-autobuilder/v3.4.5
```

## Creating the links

Now, once a release is created the links from the tag links (qa, testing, production) are not auto-created by the build pipeline. Instead, these has to be created by other means when the operators decide it's right time to deploy a certain version to a certain network. Once decided the link then must be created. This brings us to the `zos-update-worker`

The update worker is a very simple process that watches changes to `tfchain` version as updated by the Council. and apply the correct link.

Say the worker finds out that the zos version on production tfchain is set to `v3.4.5` then it will simply make sure the link from `production` is correctly pointing to the correct release tag. If not, will create that link.

On updates using motions we need to set zos3 and zos4 version and if the new release is safe to upgrade to. For example:
```json
{"safe_to_upgrade": true, "version":"v3.12.5", "version_light": "v0.1.1"}
```
- safe_to_upgrade: if the flag is set this indicates that it is safe to update all nodes in the network to this release, if not then the nodes specified in zos-config are the only ones that is getting the upgrade.

For example [here](https://github.com/threefoldtech/zos-config/blob/1a2d1339b219775ca4359535793f067630ed9062/qa.json#L68) if safe_to_upgrade is not set the only farms getting the new upgrade are the mentioned farms other farms in qa are not.

- version: is zos3 version
- version_light: is zos4(zos light) version
