# `gateway-name-proxy` type

This create a proxy with the given name to the given backends. The `name` of the proxy must be owned by a name contract on the grid. The idea is that a user can reserve a name (i.e `example`). Later he can deploy a gateway work load with name `example` on any gateway node that points to specified backends. The name then is prefix by the gateway name. For example if the gateway domain is `gent0.freefarm.com` then your full QFDN is goint to be called `example.gen0.freefarm.com`

Full name-proxy workload data is defined [here](../../../pkg/gridtypes/zos/gw_name.go)
