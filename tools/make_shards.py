import math
import sys

def make_shards(num_shards):
    """
    Return canonical shard names covering the range with the number of
    shards provided
    """
    bytes_needed = max(int(math.ceil(math.log(num_shards, 256))), 1)
    interval = ((2**64-num_shards+1) // num_shards) + 1

    end = 0
    shards = []
    shards_bin = []
    for i in range(0, num_shards):
        start = end
        end = start + interval

        if i == 0:
            shard = '-%s' % '{:016x}'.format(end)[:bytes_needed*2]
        elif i == num_shards - 1:
            shard = '%s-' % '{:016x}'.format(start)[:bytes_needed*2]
        else:
            shard = '%s-%s' % ('{:016x}'.format(start)[:bytes_needed*2], '{:016x}'.format(end)[:bytes_needed*2])
        if num_shards == 1:
            shard = '-'
        shards.append(shard)
        shards_bin.append((start, end))
    return shards, shards_bin

if len(sys.argv) != 2:
    print("Usage: python make_shards.py <num_shards_desired>")
    sys.exit(1)

print(make_shards(int(sys.argv[1])))
