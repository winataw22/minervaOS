# Runtime Tests

This tool try to help you to send bunch of test scenario to a 0-OS machine and
check the result of provisioning to see if everything goes well or not.

The test script should works out-of-box without parameter, but that's probably not
what you want, you probably want to test a specific node.

# Dependencies

The test script is a bash script which rely on a small amount of dependencies:
- `tfuser` which is a tool available in this repo
- `curl` to download response and query the api
- `jq` to parse json response

# Options

You an pass few arguments to the script:
- `-f <farmid>    specify the farm id to use`
- `-n <nodeid>    specify the node id where to provision stuff`
- `-r <target>    specify the target endpoint where sending logs`
- `-t <target>    specify the tnodb url to use to query/provision`

The farmid is only useful if you don't specify a nodeid, when you don't
have an nodeid, a random node within the farm will be used.

The nodeid is the exact node where to provison stuff.

One of the first test on the node is setting a remote redis server where
to push logs, you can customize the redis address/port with this option.

The tnodb url is the base url where to contact the mock in order to query
the api and send provision request

# Feedback

At the end of the provisioning, the test script will wait for provision
response and will display a summary of which tasks succeed or failed.
