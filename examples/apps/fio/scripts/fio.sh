cat > fio.conf <<EOF
[global]
directory=$DEVICE
size=$SIZE
direct=$DIRECT
gtod_reduce=1
time_based=1
runtime=1m
group_reporting=1

[file1]
ioengine=$IOENGINE
EOF

for rw in randread randwrite; do
    for numjobs in 1 4 8 16; do
        for iodepth in 1 4 32 128; do
            cmd="fio --rw=$$rw --bs=4K --numjobs=$$numjobs --iodepth=$$iodepth fio.conf"
            echo $$cmd
            $$cmd >> /dev/shm/pipe

            echo
            echo
            echo
        done
    done
done

for rw in read write; do
    for numjobs in 1; do
        for iodepth in 1 4 32 128 256; do
            cmd="fio --rw=$$rw --bs=1M --numjobs=$$numjobs --iodepth=$$iodepth fio.conf"
            echo $$cmd
            $$cmd >> /dev/shm/pipe

            echo
            echo
            echo
        done
    done
done
