# `ip` type
The IP workload type reserves an IP from the available contract IPs list. Which means on contract creation the user must specify number of public IPs it needs to use. The contract then will allocate this number of IPs from the farm and will kept on the contract.

When the user then add the IP workload to the deployment associated with this contract, each IP workload will pick and link to one IP from the contract.

In minimal form, `IP` workload does not require any data. But in reality it has 2 flags to pick which kind of public IP do you want

- `ipv4` (`bool`): pick one from the contract public Ipv4
- `ipv6` (`bool`): pick an IPv6 over SLAAC. Ipv6 are not reserved with a contract. They are basically free if the farm infrastructure allows Ipv6 over SLAAC.

Full `IP` workload definition can be found [here](../../../pkg/gridtypes/zos/ipv4.go)
