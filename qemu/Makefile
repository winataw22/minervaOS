all: start

prepare: kernel

zinit:
	@echo "copy zinit into overlay"
	cp $(shell which zinit) overlay/sbin/zinit

kernel:
	@echo "Download 0-OS kernel"
	wget https://bootstrap.grid.tf/kernel/net/dev.efi

start:
	bash vm.sh -n node1 -c "runmode=dev farmer_id=$(FARMERID)"
test:
	bash vm.sh -n node1 -c "runmode=test farmer_id=$(FARMERID)"

auth:
	@echo "Copying your public ssh to machine rootfs"
	mkdir -p overlay/root/.ssh
	cp ~/.ssh/id_rsa.pub overlay/root/.ssh/authorized_keys

net:
	@echo "Creating a virtual natted network"
	bash ./net.sh

run:
	@echo "Running your node"
	sudo ./vm.sh -g -n node-01 -c "farmer_id=$(id) version=v3 printk.devmsg=on runmode=dev nomodeset"
