build:
	cd ../cmds && make internet
	cd bootstrap && make release

install: build
	mkdir -p ${ROOT}/etc/zinit/
	mkdir -p ${ROOT}/bin
	mkdir -p ${ROOT}/sbin

	# install interent
	cp ../bin/internet ${ROOT}/bin

	# install bootstrap
	cp -a etc ${ROOT}
	cp -a usr ${ROOT}
	cp bootstrap/target/x86_64-unknown-linux-musl/release/bootstrap ${ROOT}/sbin/
