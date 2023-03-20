# Cloud console

- `cloud-console` is a tool to view machine logging and interact with the machine you have deployed
- It always runs on the machine's private network ip and port number equla to `20000 +last octect` of machine private IP
- For example if the machine ip is `10.20.2.2/24` this means
  - `cloud-console` is running on `10.20.2.1:20002`
- For the cloud-console to run we need to start the cloud-hypervisor with option "--serial pty" instead of tty, this allows us to interact with the vm from another process `cloud-console` in our case
- To be able to connect to the web console you should first start wireguard to connect to the private network

```
wg-quick up wireguard.conf
```

- Then go to your browser with the network router IP `10.20.2.1:20002`
