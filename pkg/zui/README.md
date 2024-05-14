# Zui module

- The zui module is a gui tool used to show the state of services and usage of the system resources.

- At start up, it shows the state of the running services until all services are up.

![image](https://github.com/threefoldtech/zos/assets/67752395/43f15965-d900-4cf6-9666-513a5b3fb847)

- After all services are up, zui shows the network information and the usage of the node resources.

![image](https://github.com/threefoldtech/zos/assets/67752395/e9186f75-2334-4c1e-a8ba-35af86eddee8)

- It can also display the errors occurred in other modules.

## Display errors

- To push errors to zui you can send a slice of errors to `PushErrors`.

### Example:

```go
zui := stubs.NewZUIStub(cl)

errors := []error{fmt.Errorf("error: failure in some module")}

lable := "Fatal"

if err := zui.PushErrors(ctx, label, errors); err != nil {
    return err
}
```

- Or you can send an empty errors slice with the label to `PushErrors` to stop displaying certain label.


## Usage

To allow gui use one of the following methods:

1. To run node in a VM using `qemu` run [vm.sh](../../qemu/vm.sh) script using `-g` flag to allow gui

```
sudo ./vm.sh -g -n node-01 -c "farmer_id=$(id) version=v3 printk.devmsg=on runmode=dev"

```

2. While the zos node is running run `zui` command in the node to open the GUI.

3. Use `alt+F2` to toggle between logs and zui

## Update zui display

To update or display more information to zui you can update the module [here](https://github.com/threefoldtech/zos/tree/main/cmds/modules/zui)

- Monitor more or less resources by updating the `resourcesRender` in `zui/prov.go`

- Update the services monitored by adding/removing services in `zui/service.go`

- Show more network configuration by updating `zui/net.go`

> **Note:** Zui will not start displaying the resources usage until all monitored services are completed.
