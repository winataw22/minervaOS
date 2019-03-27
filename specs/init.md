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
- Build our own pid 1 in rust, use the ignite as base (or as a reference)

# Discussions
After some internal discussion, runit might not be the best option due to how it was built, and the its purpose (mainly run in containers).
We are strongly leaning toward using our own init based on ignite.

