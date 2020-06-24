build_base-musl() {
    apt-get update
    apt-get install -y build-essential musl musl-tools

    echo "[+] setting up musl base system"

    # linking linux source kernel to musl path
    ln -fs /usr/include/linux /usr/include/x86_64-linux-musl/

    # linking some specific headers
    ln -fs /usr/include/asm-generic /usr/include/x86_64-linux-musl/
    ln -fs /usr/include/x86_64-linux-gnu/asm /usr/include/x86_64-linux-musl/

    # linking sys/queue not shipped by musl
    ln -fs /usr/include/x86_64-linux-gnu/sys/queue.h /usr/include/x86_64-linux-musl/sys/
}

