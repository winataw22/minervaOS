## zinit package

-zinit Package exposes function to interact with [zinit](https://github.com/threefoldtech/zinit) service life cycle management. So we can add, remove, list ... services

### Create zinit client

- To interact with zinit services you first should create an instance on zinit client

```
cl := zinit.New("/var/run/zinit.sock")
// or just use default to connect to the default socket path
cl := zinit.Default()
```

### Adding service

```
err := cl.AddService("name", zinit.InitService{
		Exec: cmd, // the command to be executed i.e the actual service
		Env: map[string]string{
			"Key": "value",
		},
        Oneshot: false,
		After:   []string{},
	})
```

### Removing a service

```
 err := cl.RemoveService(name);
```

### Loading a service so you can check its properties or do modifications

```
service, err := cl.LoadService(name)
```

### list all services

```
services, err := cl.List()
```

### Check if a specific service exists

```
exists, err := cl.Exists("name")
```

### Getting a service by name

```
cfg, err := cl.Get("name")
```

### Getting service status

```
status, err := cl.Status("name")
```

### check if the service is exited/stopped

```
status, err := cl.Status("name")
exited := status.State.Exited()

```

### Getting zinit binary version

```
version, err := cl.Version()
```

### Rebooting and shutdown the node

```
err := cl.Reboot()
err := cl.Shutdown()
```

### Starting/Stopping a service

```
err := cl.Start("name")
err := cl.stop("name")
```

### Start/Stop a service and wait for it to start

```
err := cl.StartWait(time.Second*20, "name");
err := cl.StopWait(time.Second*20, "name");
```

### Start monitoring a service

```
err := cl.Monitor("name");
```

### Forget a stopped service

```
err := cl.Forget("name");
```

### Kill a running service

```
err := cl.Kill("name", zinit.SIGTERM); // or choose whatever signal you want
```

### Starting/Stopping multiple services

```
err := cl.StartMultiple(time.Second*20, "name1", "name2", ...., "namex")
err := cl.StopMultiple(time.Second*20, "name1", "name2", ...., "namex")
```

### List services that matches some filters

```
filter1 := zinit.WithExec("udhcpc")
filter2 := zinit.zinit.WithName(dhcp)
matched, err := s.z.Matches(filter1, filter2)
```

### Destroy a service (Destroy given services completely (stop, forget and remove))

```
err := cl.Destroy(20*time.Second, "name");
```
