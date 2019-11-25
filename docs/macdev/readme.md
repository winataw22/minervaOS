# MacOS Developer

0-OS (v2) uses a Linux kernel and is really build with a linux environment in mind.
As a developer working from a MacOS environment you will have troubles running the 0-OS code.

Using [Docker][docker] you can work from a Linux development environment, hosted from your MacOS Host machine.
In this README we'll do exactly that using the standard Ubuntu [Docker][docker] container as our base.

## Setup

0. Make sure to have Docker installed, and configured (also make sure you have your code folder path shared in your Docker preferences).
1. Start an _Ubuntu_ Docker container with your shared code directory mounted as a volume:
```bash
docker run -ti -v "$HOME/oss":/oss ubuntu /bin/bash
```
2. Make sure your environment is updated and upgraded using `apt-get`.
3. Install Go (`1.13`) from src using the following link or the one you found on [the downloads page](https://golang.org/dl/):
```bash
wget https://dl.google.com/go/go1.13.3.linux-amd64.tar.gz
sudo tar -xvf go1.13.3.linux-amd64.tar.gz
sudo mv go /usr/local
```
4. Add the following to your `$HOME/.bashrc` and `source` it:
```vim
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
```
5. Confirm you have Go installed correctly:
```
go version && go env
```
6. Go to your `zos` code `pkg` directory hosted from your MacOS development machine within your docker `/bin/bash`:
```bash
cd /oss/github.com/threefoldtech/zos/pkg
```
7. Install the dependencies for testing:
```bash
make getdeps
```
8. Run tests and verify all works as expected:
```bash
make test
```
9. Build `zos`:
```bash
make build
```

If you can successfully do step (8) and step (9) you
can now contribute to `zos` as a MacOS developer.
Testing and compiling you'll do from within your container's shell,
coding you can do from your beloved IDE on your MacOS development environment.

[docker]: https://www.docker.com
