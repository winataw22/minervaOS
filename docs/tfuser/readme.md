# tfuser CLI

tfuser is a small CLI tool that we created to help developer deploy workload on a 0-OS node during development.

It has 3 main commands:

- **id**: use to generate user identity
- **generate**: used to provisioning reservation schema
- **provision**: used to send schema created with the generate command to a node

## Examples

### Generate a network provisioning schema

Here is a full example how you would use tfuser to create a network schema.

1. create a network from scratch
`tfuser generate --schema network.json network create --node {nodeID}`

the `--node` flag point to the node ID of the exit node we have chosen for the network.  
the value of the `--schema` flag is a path where the provisoin schema will be written by tfuser. If not specified the schema will be printed to stdout.

2. add another node to the network:
`tfuser generate --schema network.json network add-node --node {nodeID}`

the `--node` flag point to the node ID we want to add to the network

The result of these two command should looks like:
```json
{
  "id": "",
  "user_id": "E6Zb6cLHYuuXm1KSMeYc1DYNKfknGQqFVKS7tMc3mBvJ",
  "type": "network",
  "data": {
    "network_id": "ZKhTrpy7tubzG5fxqdsWzwoXbEhtwJ1yxDBz8xDUf9J",
    "resources": [
      {
        "node_id": {
          "id": "qzuTJJVd5boi6Uyoco1WWnSgzTb7q8uN79AjBT9x9N3",
          "farmer_id": "CemYjciEmuvYVKDFXYaZLdGsCdLDRp4U1Xu1LPPrQNkK",
          "reachability_v4": "public",
          "reachability_v6": "public"
        },
        "prefix": "2a02:1802:5e:1001::/64",
        "link_local": "fe80::1001/64",
        "peers": [
          {
            "type": "wireguard",
            "prefix": "2a02:1802:5e:1001::/64",
            "Connection": {
              "ip": "2a02:1802:5e:0:1000:0:ff:1",
              "port": 4097,
              "key": "tDbiZ+1kng5fhRbjTQanmF83ETtvj1Rs5AG1fiOh+RQ=",
              "private_key": "66e445cb705e14788f01cff5355869eafa2ec70c9ea8bb9c4cf32557f683770e3c5125b6e865afad1d0593f0d6c54887275452b1f0db4429d23aeca729ed5dec37ea74edf62df099017cb0bec49c1929dce7bc793018491afb6b8918"
            }
          },
          {
            "type": "wireguard",
            "prefix": "2a02:1802:5e:153f::/64",
            "Connection": {
              "ip": "<nil>",
              "port": 5439,
              "key": "UOqDqmhynYYRmn0E1tMP+176ApHH33tyZTO7Br49LBg=",
              "private_key": "b0f3fb4d599e41b459c011ea2057348501e7a22f6be7a21d5e82cb3ce439ee57c2a231ffcc62cdc71eb42e8336a4b404146fa28b124d02c19216271cead7e3a5d096b730eb5190fe8a655340a78fa403053f22a4dd0548f79da777f4"
            }
          }
        ],
        "exit_point": 1
      },
      {
        "node_id": {
          "id": "37zg5cmfHQdMmzcqdBR7YFRCQZqA35wdEChk7ccR4tNM",
          "farmer_id": "CemYjciEmuvYVKDFXYaZLdGsCdLDRp4U1Xu1LPPrQNkK",
          "reachability_v4": "hidden",
          "reachability_v6": "hidden"
        },
        "prefix": "2a02:1802:5e:153f::/64",
        "link_local": "fe80::153f/64",
        "peers": [
          {
            "type": "wireguard",
            "prefix": "2a02:1802:5e:1001::/64",
            "Connection": {
              "ip": "2a02:1802:5e:0:1000:0:ff:1",
              "port": 4097,
              "key": "tDbiZ+1kng5fhRbjTQanmF83ETtvj1Rs5AG1fiOh+RQ=",
              "private_key": "66e445cb705e14788f01cff5355869eafa2ec70c9ea8bb9c4cf32557f683770e3c5125b6e865afad1d0593f0d6c54887275452b1f0db4429d23aeca729ed5dec37fa74edf62df099017cb0bec49c1929dce7bc793018491afb6b8918"
            }
          },
          {
            "type": "wireguard",
            "prefix": "2a02:1802:5e:153f::/64",
            "Connection": {
              "ip": "<nil>",
              "port": 5439,
              "key": "UOqDqmhynYYRmn0E1tMP+176ApHH33tyZTO7Br49LBg=",
              "private_key": "b0f3fb4d599e41a459c011ea2057348501e7a22f6be7a21d5e82cb3ce439ee57c2a231ffcc62cdc71eb42e8336a4b404146fa28b124d02c19216271cead7e3a5d096b730eb5190fe8a655340a78fa403053f22a4dd0548f79da777f4"
            }
          }
        ],
        "exit_point": 0
      }
    ],
    "prefix_zero": "2a02:1802:5e::/64",
    "exit_point": {
      "ipv4_conf": null,
      "ipv4_dnat": null,
      "ipv6_conf": {
        "addr": "fe80::1000:1/64",
        "gateway": "fe80::1",
        "metric": 0,
        "iface": "public"
      },
      "ipv6_allow": []
    },
    "allocation_nr": 0,
    "version": 1
  },
  "signature": "",
  "created": "",
  "duration": 0,
}
```


### Generate a container provisioning schema

```shell
tfuser generate --schema ubuntu.json container \
--flist https://hub.grid.tf/tf-official-apps/ubuntu-bionic-build.flist \
--corex \
--entrypoint /bin/bash \
--network ZKhTrpy7tubzG5fxqdsWzwoXbEhtwJ1yxDBz8xDUf9J \
--env KEY:VALUE
```

### Provision a workload on a node

To provision a workload on a node, you need to have a user identity. It is used to sign the schema before sending to the node.  
Generate an identity using `tfuser id`. This command will generate a `user.seed` file that you need to use with the provision command.


```shell
tfuser provision --schema container.json --duration 2 --seed user.seed --node {nodeID}
```