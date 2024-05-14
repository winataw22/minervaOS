# Environment module

This module is used to get information about the running environment of a node.

It is also used to get the organization default configurations defined [here](https://github.com/threefoldtech/zos-config)
according to the running mode.

## Usage

To get information about the running environment of the node you can use one of the following:

1. use `environment.Get()` to get the information if possible.

2. use `environment.MustGet()` to get information or panic on error.

To get organization defined configurations:

1. use `environment.GetConfig()` to get the organization configurations for the running mode.

2. use `environment.GetConfigForMode(mode)` to get configurations for a specific mode.

### Usage Example

```go
env, err := environment.Get()
if err != nil {
    log.Fatal().Err(err).Msg("could not get information about running environment")
}

config, err := environment.GetConfigForMode(env.RunningMode)
if err != nil {
    log.Fatal().Err(err).Msgf("could not get configurations of mode %s", env.RunningMode)
}

```
