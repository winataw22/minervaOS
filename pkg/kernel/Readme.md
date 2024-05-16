# Kernel Package

This package exposes functions used to parse and get parameters passed to the kernel on boot.

## Usage

### Load kernel parameters:

```go
// params will be used to get any parameter passed to the kernel.
params := kernel.GetParams()
```

### Check if a parameter exist:

```go
if !params.Exists("something") {
    log.Fatal().Msg("`something` is not set")
}
```

### Get values linked to a key:

```go
if values, ok := params.Get("something"); ok {
    if len(values) != 1 {
        log.Fatal().Msg("something values should be of length 1")
    }

    if values[0] != "one thing" {
        log.Fatal().Msg("something value is wrong")
    }
} else {
    log.Fatal().Msg("`something` is not set")
}
```

### Gets a single value for given key:

```go
if value, ok := params.GetOne("something"); !ok {
    log.Fatal().Msg("`something` is not set")
}

```

### Checks if zos-debug is set:

 ```go
if ok := params.IsDebug(); ok {
    log.Debug().Msg("zos is running in debug mode")
}
 ```

### Checks if gpu is disabled:

 ```go
if ok := params.IsGPUDisabled(); ok {
    log.Debug().Msg("GPU is disabled")
}
 ```

### Checks if zos-debug-vm is set:

 ```go
if ok := params.IsVirtualMachine(); ok {
    log.Debug().Msg("zos thinks it's running on virtual machine")
}
 ```
