# Reservation of capacity on the grid

Reservations of capacity on the grid now require payments. If the user wishes to [reserve some capacity](it_contract.md#reservation), he would register a reservation to the explorer. As a result he will receive a payment invitation. When the user transfers the full amount due, the reservation will be deployed on the grid. If the reservation fails for any reason, the user will be refunded.

The payment of reservation will be done with Stellar wallets. We issued TFT on the Stellar network and we expect the user to pay with these assets. We support multiple assets for both testing and production.

## Supported assets

Network         | Asset Code    | Asset Issuer |
| ------------- | ------------- | ------------- |
| Production    | TFT           | GBOVQKJYHXRR3DX6NOX2RRYFRCUMSADGDESTDNBDS6CDVLGVESRTAC47
| Testnet       | TFT           | GA47YZA3PKFUZMPLQ3B5F2E3CJIB57TGGU7SPCQT2WAEYKN766PWIMB3
| Production    | FreeTFT           | GCBGS5TFE2BPPUVY55ZPEMWWGR6CLQ7T6P46SOFGHXEBJ34MSP6HVEUT
| Testnet       | FreeTFT           | GBLDUINEFYTF7XEE7YNWA3JQS4K2VD37YU7I2YAE7R5AHZDKQXSS2J6R

For More information: [https://github.com/threefoldfoundation/tft-stellar](https://github.com/threefoldfoundation/tft-stellar)

## Prerequisites

1. A Stellar wallet with funds.
2. A trustline to the asset in which the user like to do the payment.

# Stellar Wallet

A Stellar wallet is ought to have funds of one of the supported assets and the native currency XLM. The reason the wallet needs these XLM is because transaction created on the stellar network require a fee paid in the native currency XLM. 

Currently we support Stellar wallets in Jumpscale: [https://github.com/threefoldtech/jumpscaleX_libs/tree/unstable/JumpscaleLibs/clients/stellar](https://github.com/threefoldtech/jumpscaleX_libs/tree/unstable/JumpscaleLibs/clients/stellar).

## Stellar Trustlines

> Skip this step if you already have balances of on the supported assets or a trustline is already set.

In order to support a wallet to hold any of the supported assets a trustline is required. A trustline means that you trust the issuer of said asset.

Example of setting a trustline to `Production TFT`

```python
# valid types for network: STD and TEST, by default it is set to STD
JSX> wallet = j.clients.stellar.new('my_wallet', network='STD', secret='S.....')
# available as `j.clients.stellar.my_wallet` from now on

JSX> wallet.add_trustline('TFT','GBOVQKJYHXRR3DX6NOX2RRYFRCUMSADGDESTDNBDS6CDVLGVESRTAC47')
```

After setting the trustline you can receive `TFT` from issuer `GBOVQKJYHXRR3DX6NOX2RRYFRCUMSADGDESTDNBDS6CDVLGVESRTAC47`. 

# Making a reservation

Now that we have a wallet with funds and trustlines we can create a reservation.
We will use the ZosV2 sal in Jumpscale to create and pay for a reservation.

Example: 

```python
JSX> wallet = j.clients.stellar.get(name="my_wallet", network="STD")

JSX> import time
JSX> zos = j.sal.zosv2
# create a reservation
JSX> r = zos.reservation_create()

JSX> zos.volume.create(r, "72CP8QPhMSpF7MbSvNR1TYZFbTnbRiuyvq5xwcoRNAib", size=1, type='SSD')
JSX> expiration = j.data.time.epoch + (3600 * 24 * 365)
# register the reservation
JSX> registered_reservation = zos.reservation_register(r, expiration)
JSX> time.sleep(5)
# inspect the result of the reservation provisioning
JSX> result = zos.reservation_result(registered_reservation.reservation_id)
JSX> print(result)
# payout farmer
JSX> zos.billing.payout_farmers(wallet, registered_reservation)
```