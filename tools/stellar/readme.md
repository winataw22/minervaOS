## Stellar client for Multisignature transaction

This client is developed for the sole purpose of recovering escrow accounts that are for some reason not refunding nor paying farmers.

## Prerequisites

1. Start the explorer with the 5 multisig wallets (all of them should be on the same network and have some funds)
2. Ask the client for the escrow account address
3. Ask the client for the amount that needs to be recovered or search it on the Stellar transaction explorer.
4. Select a destination wallet

## Usage

1. Create the multisig transaction 

```
stellar create --seed "multisigwalletseed" --network "somenetwork" --asset "someasset" --destination "somedestination" --from "escrowaccountaddress" --amount "someamountasstring"
```

This will output something like:

```
Transaction to be signed: AAAAAPODclmCjkbWZYnoAPFTywzsVcd0T0V8nUogz3LFlya0AAAAZAAPNm8AAAAIAAAAAQAAAAAAAAAAAAAAAF6EkEQAAAAAAAAAAQAAAAAAAAABAAAAALX7uq+eXcgHVVKPAjAjscsoT2lnDH4ucBIuB6toxeoiAAAAAVRGVAAAAAAAOfxkG3qLTLHrhsPS6JsSUB7+ZjU/J4oT1YBMKb/3n2QAAAAABfXhAAAAAAAAAAABQANAbAAAAEAFPX5v7RyZ8quNt/eWN+CEp/3JQvg6bP2ncxNbO/6w2vvoav/K2SuHeP+Ur1ZEjuKOEOA6tQK43X+JKQEINEca
```

2. Make 2 out of 5 other multisig wallet sign this transaction

In the first step 1 out of 5 wallets already signs the transaction and we need a minimum of 3 out of 5 signatures to complete a transaction. So in this case 2 out of 5 need to sign!

```
stellar sign --seed "multisigwalletseed" --network "somenetwork" --transaction 'AAAAAPODclmCjkbWZYnoAPFTywzsVcd0T0V8nUogz3LFlya0AAAAZAAPNm8AAAAIAAAAAQAAAAAAAAAAAAAAAF6EkEQAAAAAAAAAAQAAAAAAAAABAAAAALX7uq+eXcgHVVKPAjAjscsoT2lnDH4ucBIuB6toxeoiAAAAAVRGVAAAAAAAOfxkG3qLTLHrhsPS6JsSUB7+ZjU/J4oT1YBMKb/3n2QAAAAABfXhAAAAAAAAAAABQANAbAAAAEAFPX5v7RyZ8quNt/eWN+CEp/3JQvg6bP2ncxNbO/6w2vvoav/K2SuHeP+Ur1ZEjuKOEOA6tQK43X+JKQEINEca
```

Repeat until nothing is returned! 