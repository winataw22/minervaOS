# Upgrade module

The upgrade module is the one responsible to keep a 0-OS node always up to date.

It checks the network for new releases of 0-OS modules.

When a new releases is found, it will download the flist containing the new version of the module.

Then copy the new module in place, execute the migration scripts if there are any. Then restart the updated module with the new binaries.

## Publisher

The upgrade module implement the Publisher interface
```go
// Publisher is the interface that define how the upgrade are published
type Publisher interface {
	// Get retrieve the Upgrade object for a specific version
	Get(version semver.Version) (Upgrade, error)
	//Latest return the latest version available
	Latest() (semver.Version, error)
	// List all the version this publisher has
	List() ([]semver.Version, error)
}
```

This interface define how the module get information about new releases.

As for now, the module only implement an HTTP publisher. The HTTP publisher relies on an HTTP server to get information.
Here is a description of what is expected from the HTTP server:

Imagine the HTTP publisher has a base URL of: `https://releases.grid.tf`

It needs to expose 3 endpoints:
- GET `https://releases.grid.tf/versions`: return a list of all the versions this publisher knows about, example:

```json
[
    "0.0.1",
    "0.0.2",
    "0.0.3",
    "0.1.0",
    "0.1.1"
]
```
- GET `https://releases.grid.tf/latest` return the latest version, example:

```json
"0.1.1"
```

- GET `https://releases.grid.tf/{versions}` : return the upgrade object for this version, example for `https://releases.grid.tf/0.0.1`:

```json
{
    "flist":"https://hub.grid.tf/tf-official-apps/threefoldtech-0-db-release-1.0.0.flist",
    "transaction_id":"",
    "signature":"e5b2cab466e43d8765e6dcf968d1af9e"
}
```
