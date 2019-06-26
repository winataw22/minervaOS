
## Services Boot Sequence
- node-ready
  - storaged
  - netwokrd
- boot
  - contd
  - flister
  - upgrade

## Pseudo boot steps
both node-ready and boot are not actual services, but instead they are there to define a `boot stage`. for example once `node-ready` service is (ready) it means all crucial system services defined by 0-initramfs are now running. This now only contains the `udev` daemons, but it's easy to add more without the need to change all the other services.

`boot` service is similar, but guarantees that some zos2 services are running (for example `storaged`), before starting other services like flister which requires storaged
