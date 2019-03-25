# Inter-process communication
According to the quick architecture discussion session we will have sub processes that
are responsible for certain tasks for example `networking`, or `vm`. The api should be
able to communicate with the individual solutions to `pull` a solution by combining functionality
from the separate components.

A sort of secured, inter process communication is needed, where the api can reach the different components
also, handle different signals when an event that require attention. For example a `network` manager needs
to free up a virtual network device when the VM that uses it exits.

We did some research for the best inter-process bus solution we can have, but unfortunately the available solution didn't
fit precisely (for exampele DBus, and gRPC). Hence we built our own solution [zbus](zbus.md)

## Overview
The API will receive a DSL that describes a certain service. For example a container. While the dsl is not specified yet, we can have this
pseudo dsl script
```yaml
- container:
    name: container-1
    image: image/id
    require:
        storage:
            # storage specs
        network:
            # network specs
```

The api, will use the bus to find out who implements the `network` api, and doing the required calls on the network component (over zbus),
then just passing the results to the container component, to do it's part.

In this scenario, the API is a DSL interpreter and a broker that can ask proper components for an object.