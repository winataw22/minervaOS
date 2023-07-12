all: start

prepare: kernel

zinit:
	@echo "copy zinit into overlay"
	cp $(shell which zinit) overlay/sbin/zinit

kernel:
	@echo "Download 0-OS kernel"
	wget https://bootstrap.grid.tf/kernel/zero-os-development-zos-v3-generic-0ad6c11534.efi

start:
	bash vm.sh -n node1 -c "runmode=dev farmer_id=$(FARMERID)"
test:
	bash vm.sh -n node1 -c "runmode=test farmer_id=$(FARMERID)"
