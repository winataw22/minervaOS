# explorer

## managing encrypted seeds for escrow accounts

The seeds of the escrow accounts are encrypted with a key based on the seed used
to start the explorer. This means that changing this seed will cause decryption
of these seeds, and thus their usage by the explorer, to fail. If for any reason
the seed used to start the explorer changes, the operator will need to clear existing
escrow accounts and their associated seeds. To this end, the explorer can be restarted
with the `-flush-escrows` flag. When this flag is passed, confirmation will be asked
on the command line if a user really wants to remove this data from the db. If the
operator changes his/her mind, the explorer will exit, and needs to be restarted
without this flag.

### disposing of encrypted seeds

It is possible, that the addresses used by an escrow are currently active, i.e.
a user has created a reservation and is in the process of paying for it. Although
it is technically possible to swap the addresses in the escrow, the user will still
try to pay to the old address, so this case can't really be handled. In order to not
lose funds however, it is encouraged to back up the accounts before they are removed.
If the explorer is started with the multisig feature enabled by providing sufficient
backup signers, the funds (if any) on the escrow address can still be recovered, and
returned to their rightfull owners, by creating multisigs with the backup signers
for the addresses. Note that the public addresses are not encrypted, as such, even
if the seed used to start the explorer is lost completely, the escrow funds can still
be recovered

