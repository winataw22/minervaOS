build:
	cd ../cmds && make internet

install: build
	mkdir -p ${ROOT}/etc/zinit/
	mkdir -p ${ROOT}/bin
	cp -a etc ${ROOT}
	cp ../bin/internet ${ROOT}/bin
	cp ../zinit/internet.yaml ${ROOT}/etc/zinit/