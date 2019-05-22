all: start

prepare: kernel zinit

zinit:
	@echo "copy zinit into overlay"
	cp $(shell which zinit) overlay/sbin/zinit

kernel:
	@echo "Download 0-OS kernel"
	wget https://bootstrap.grid.tf/kernel/zero-os-development-zos-v2-generic.efi -O zosv2.efi
	ln -sf zosv2.efi zos.efi

start:
	bash vm.sh -n node1
