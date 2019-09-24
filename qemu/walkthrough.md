# 0-OS Walkthrough

## Create your identity

In order to identify yourself, you'll need a unique key.
You can easily generate that key with `tfuser` tool.

```
tfuser id -o /tmp/demo-user.seed
```

## Register your Farm

In order to group your nodes and identify them to be your, you need
to add them into your farm. But first, you need to create your farm.

In order to create your farm, you'll need your seed you just created.
This is the only way to idenfity and know you're the owner of the farm.

```
tffarm farm register --seed /tmp/demo-user.seed MyNewFarm
```

## Start your node

Start your VM with the `farmer_id` kernel argument.
If you're using the makefile, just do: `make FARMERID=.... start`
