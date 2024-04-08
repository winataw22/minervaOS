# Upgrade module

The upgrade module is responsible to keep a zos node always up to date.

It checks the hub for new releases of zos packages.

If a new release is available, it will then update the packages and restart the updated module with the new binaries if required.

## Usage

To run the upgrade module you first need to create a new upgrader instance with the supported options.

|     Option     |                 Description                  |        Default        |
| :------------: | :------------------------------------------: | :-------------------: |
| `NoZosUpgrade` | enable or disable the update of zos binaries |  enabled by default   |
|   `Storage`    |    overrides the default hub storage url     |     `hub.grid.tf`     |
|    `Zinit`     |      overrides the default zinit socket      | "/var/run/zinit.sock" |

```go
upgrader, err := upgrade.NewUpgrader(root, upgrade.NoZosUpgrade(debug))
if err != nil {
    log.Fatal().Err(err).Msg("failed to initialize upgrader")
}
```

Then run the upgrader `upgrader.Run(ctx)`

## How it works

The upgrader module has two running modes depeding on the booting method.

### Bootstrap Method

Running the upgrader on a node run with `bootstrap` will periodically check the hub for latest tag,
and if that tag differs from the current one, it updates the local packages to latest.

If the update failed, the upgrader would attempts to install the packages again every `10 secounds` until all packages are successfully updated to prevent partial updates.

The upgrader runs periodically every hour to check for new updates.

### Other Methods

If the node is booted with any other method, the required packages are likely not installed.
The upgrader checks if this is the first run, and if so, it installs all the latest packages and blocks forever.
