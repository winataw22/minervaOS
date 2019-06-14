set -x

mount -t tmpfs cgroup_root /sys/fs/cgroup

subsys="cpuset cpu cpuacct blkio memory devices freezer net_cls perf_event net_prio"

for sys in $subsys; do
    mkdir -p /sys/fs/cgroup/$sys
    mount -t cgroup $sys -o $sys /sys/fs/cgroup/$sys/
done