# `zlogs` type

Zlogs is a utility workload that allows you to stream `zmachine` logs to a remote location.

The `zlogs` workload needs to know what `zmachine` to stream logs of and also the `target` location to stream the logs to. `zlogs` uses internally the [`tailstream`](https://github.com/threefoldtech/tailstream) so it supports any streaming url that is supported by this utility.

`zlogs` workload runs inside the same private network as the `zmachine` instance. Which means zlogs can stream logs to other `zmachines` that is running inside the same private network (possibly on different nodes).

For example, you can run [`logagg`](https://github.com/threefoldtech/logagg) which is a web-socket server that can work with `tailstream` web-socket protocol.

Check `zlogs` configuration [here](../../../pkg/gridtypes/zos/zlogs.go)
