# Inter-process communication
According to the quick architecture discussion session we will have sub processes that
are responsible for certain tasks for example `networking`, or `vm`. The api should be 
able to communicate with the individual solutions to `pull` a solution by combining functionality
from the separate components.

A sort of secured, inter process communication is needed, where the api can reach the different components
also, handle different signals when an event that require attention. For example a `network` manager needs
to free up a virtual network device when the VM that uses it exits. 

I did some research on the matter, and I found the [`DBus`](https://www.freedesktop.org/wiki/Software/dbus/) solution really powerful, and easy to use. A nice GO binding is available [here](https://github.com/godbus/dbus).

I experimented with the `dbus` binding in go with good results. A server need to define an interface, then provide the implementation
for that interface. Calling from other apps is easy once you know the bus-name, object, and method signature you want to call.

## Overview 
WIP...