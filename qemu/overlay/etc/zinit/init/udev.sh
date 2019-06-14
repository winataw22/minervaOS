#!/bin/sh

set -ex

udevadm trigger --action=add
udevadm settle