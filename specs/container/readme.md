# Container module

This module is responsible to manage containerized applications.
Its only focus is about starting a process with the proper isolation. So there is no notion of storage or networking in this module, this is handled by the layer above.

The container runtime used will be compatible with the [OCI specification](https://github.com/opencontainers/runtime-spec), but which one in particular is still to be decided.

### Interface

```go
type Container struct{
    ID string
    // ...TODO
}

type ContainerConfig struct {
    // Path to the root of the filesystem bundle of the container
    RootFS string
    // Volume to mount bind in the container
    Volume map[string]string
    // Environment variable to set in the container
    Env map[string]string
}

type ContainerModule interface {
    // Create creates a container
    Create(config ContainerConfig) (c Container, err error)
    // Delete deletes any resources held by the container
    Delete(ID string) (error)
    //create and run a container
    Run(config ContainerConfig) (c Container, err error)
    //executes the user defined process in a created container
    Start(ID string) (error)
    // pause suspends all processes inside the container
    Pause(ID string) (error)
    // resumes all processes that have been previously paused
    Resume(ID string) (error)
    // Ps displays the processes running inside a container
    Ps(ID string) (error)
}
```