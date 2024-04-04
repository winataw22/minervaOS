#!/bin/sh
set -e

vol=$1

# The given volume can either be an overlay mount
# or an actual btrfs subvol.
# in case of an overlay we need to extract the upperdir path
if options=$(findmnt $vol -t overlay -r -n -o OPTIONS); then
  # extract the upperdir path
  vol=$(echo $options | cut -d ',' -f 4 | cut -d '=' -f 2)
  vol=${vol%"/rw"}
fi

if ! info=$(btrfs subvol show $vol 2>/dev/null); then
  echo "invalid btrfs volume '$vol'"
  exit 1
fi

# now we hopefully sure we have a path to an actual btrfs subvol
# we then try to extract the quota information
id=$(echo "$info" | grep 'Subvolume ID:'| cut -f 4)
output=$(btrfs qgroup show --raw -r $vol| grep "^0/$id")

size=$(echo $output | cut -d ' ' -f 4)
used=$(echo $output | cut -d ' ' -f 3)

echo $size $(( $size - $used ))
