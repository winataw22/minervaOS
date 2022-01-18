#!env sh

# This file has some initialization steps that should be executed after
# the node basic services are loaded (as defined by 0-initramfs), but before
# the rest of the system is booted.


setup_loopback() {
    ip link set lo up
}

disable_overlay() {
    rmmod ata_piix
    rmmod pata_acpi
    rmmod ata_generic
    rmmod libata
    partprobe
}

main() {
    # bring the loop back interface up
    setup_loopback

    # does not expose qemu overlay to the system
    disable_overlay

    exit 0
}

main
