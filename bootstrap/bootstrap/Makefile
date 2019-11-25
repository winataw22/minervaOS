
release: prepare
	cargo build --release --target=x86_64-unknown-linux-musl --features vendored

debug: prepare
	cargo build --target=x86_64-unknown-linux-musl --features vendored

test: prepare
	cargo test --target=x86_64-unknown-linux-musl --features vendored

prepare:
	rustup target  add x86_64-unknown-linux-musl
