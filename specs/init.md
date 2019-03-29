# PID 1 (init)
The init process is very important to the system, and this process specifically can not be restart. Hence it also can't get life updated.
This means a PID1 in zos must do minimal tasks as follows:

- Start configured system processes
  - udev, syslogd, klogd, redis, haveged, etc...
  - API, and separate modules
- init, must make sure all theses services are always up and running, re-spawn if needed
- networking ? (may be this should delegated to a separate module)
- shutdown, and reboot

# Available options on the market
- [`runit`](http://smarden.org/runit/) (very light and configurable)
- `systemd` (too much)
- `ignite` (written in rust, pretty immature and no active development)
- [finit](https://github.com/troglobit/finit) :  http://troglobit.com/finit.html
- Build our own pid 1 in rust, use the ignite as base (or as a reference)

# Discussions
After some internal discussion, runit might not be the best option due to how it was built, and the its purpose (mainly run in containers).
We are strongly leaning toward using our own init based on ignite.

# Implementation proposal
- Once the init process starts it loads all services configurations
- Configuration is analysed for dependencies cycles, to avoid blocking
- Once configuration is validated, a `job` thread is started for each defined service
- The `job` thread will check dependencies state reported by other thread services, once they are all `ready` it will spawn
it's own service, make sure it's always running by re-spawning (if needed), a `oneshot` service will never re-spawn.
- A service status can be one of the following
  - running
  - ok (only `oneshot` can be in this state)
  - error (a `job` can not start because one of it's dependencies failed or the binary does not exist)
  - re-spawn (process is being re-spawned)
- All services logs are written directly to kernel ring-buffer
  - Optionally later on, one of the daemon can be responsible of reading the logs and push them somewhere else.
- once a service update it's status, other `waiting` threads (that depends on this one) will get freed to take start.
## Controlling:
A unix socket interface can be used to control the init process as follows
- Shutdown, Reboot:
  - the manager, will set global runlevel to shutdown, ask individual services to die.
  - once each service exits, their monitor threads will not re-spawn due to global runtime state
  - once all services are down, a shutdown (or reboot) is performed.
- Status inquiry
  - List all configured services and their status.

> Do we need to support starting and stopping individual services ? with start/stop commands ?