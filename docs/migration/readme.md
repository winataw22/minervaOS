# Migration to latest V2 version

## From V1
Unfortunately V2 is not compatible with V1 because the entire reservation process has been updated, hence there is noway to be backward compatible.

To migrate to V2, you need to simply build a boot image as per the [bootstrap](https://bootstrap.grid.tf) then boot your node with this new image.

Because there is no compatiblity wih V1, ZOS V2 will reuse the disks and apply the new disk layout which means **<span style="color: red">ALL THE DATA ON THE NODE WILL BE DELETED</span>**