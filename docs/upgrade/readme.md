# Upgrade module

## Zbus

Upgrade is an autonomous module and is not reachable over zbus.

## Home Directory
upgrade keeps some data in the following locations
| directory | path|
|----|---|
| root| `/var/cache/modules/upgrade`|


## zinit unit

Upgrade module depends on network and flist module. This is because is requires network connection to check for new update and the flist module to download the upgrade flist on the node.

## Philosophy

0-OS is meant to be a black box no one can access. While this provide some nice security features it also makes it harder to manage. Specially when it comes to update/upgrade.

Since 0-OS is meant to be driven by some blockchain transaction, the version of the software running on 0-OS could also be driven by a consensus of technical people agreeing on a new version to deploy.

Once a new version of a component is ready, a multisig transaction is created by the 0-OS team and X number of person needs to sign the transaction to approve the deployment of the new version.

Using a blockchain as a source for upgrade gives us the ability to create different version of the grid. Version matching the different network of the blockchain:

- Dev: ephemeral network only setup to develop and test new features. Can be created and reset at anytime
- Test: Mostly stable feature that needs to be tested at scale, allow preview and test of new features. Always the latest greatest. This network can be reset sometimes, but should be relatively stable.
- Main: Released of stable version. Used to run the real grid with real money. Cannot be reset ever. Only stable and battle tested feature reach this level.

Since the upgrade will be triggered by a transaction on the blockchain, building a network of node using the same version is as simple as just pointing which network of the blockchain to watch.

The upgrade transaction should contains a description and signature of all the new component to install. This allow to ensure the binaries installed on the node are an exact copy of what as been announced on the blockchain.

0-OS could even periodically verify that all the running modules are not corrupted or modified by comparing the hash of the binary to the content of the blockchain.

## Technical

0-OS is designed to provide maximum uptime for its workload, rebooting a node should never be required to upgrade any of its component (except when we push a kernel upgrade).

The only way to get code onto a 0-OS is using flist. An upgrade flist will be composed of the new binary to install and in some case a migration script.

![upgrade flow](../../assets/0-OS_upgrade_flow.png)

### Flist upgrade layout

The files in the upgrade flist needs to be located in the filesystem tree at the same destination they would need to be in 0-OS. This allow the upgrade code to stays simple and only does a copy from the flist to the root filesystem of the node.

Some hooks scripts will be executed during the upgrade flow if there are present in the flist. These files needs to be executable, be located at the root of the flist and named:

- pre-copy
- post-copy
- migrate
- post-start

Example:

0-OS filesystem:

```
root
├── bin
    ├── containerd
    └── runc
```

upgrade flist:

```
root
├── bin
│   ├── containerd
│   └── flist_module_0.2.0
├── etc
|   └── containerd
|       └── config.toml
├── migrate
├── post-copy
├── post-start
└── pre-copy
```

After upgrade:

```
root
├── bin
│   ├── containerd
│   ├── flist_module_0.2.0
│   └── runc
└── etc
    └── containerd
        └── config.toml
```

### Upgrade watcher

This component is going to be responsible to watch new upgrade being publish on the blockchain. He's also going to be the one driving the upgrade. Its responsibilities will be:

- watch upgrade publication
- schedule upgrade
  - it always needs to aim for a minimal to no downtime if possible.
  - if some downtime is required, arrange to make it during a low traffic hour to impact as less as possible the users.
- in the event of the cache being corrupted, it will need to re-downloads all the component requires to run the workload present on the node. Some workload might still required previous version of some component, so during re-population of the cache it needs to make sure to grab all the versions required.

In practice the actual upgrade watcher doesn't have the logic to schedule upgrade yet. It will directly apply the upgrade as soon as it finds it. This will be improved in combing versions
