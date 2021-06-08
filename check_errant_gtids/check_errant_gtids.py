#!/usr/bin/python3
# pylint: disable=fixme, line-too-long, invalid-name, multiple-statements
#
# Tool to check for the presence of errant GTIDs in a Vitess keyspace
#
# Usage:
#   check_errant_gtids.py <vtctld_host>:<vtctld_port> <keyspace>
#
#   vtctlclient is expected to be in the PATH
#
#   Will:
#     * Contact vtctld and enumerate the shards in the keyspace
#     * Walk each shard, determining heuristically if there appears to be
#       any errant GTIDs
#
#   Note:
#     * Not expected to function correctly when failovers are in progress
#     * If ANY of the shard members are down, we will probably just timeout
#       and short-circuit the test
#

from itertools import permutations
import json
import os
import shutil
import subprocess
import sys
import time


def run_command(command, timeout=None, shell=False, path=None):
    '''Runs a command as a subprocess
       Returns: exitcode, stderr, stdout
       stderr and stdout are converted from bytestreams, interpreted as UTF8
       command - string for command to run
       timeout - number of seconds to wait while running command
       shell - False: launch process directly;  True: launch in a subshell
       path - Prepend PATH variable in subprocess environment with provided
              path
       On timeout, raises subprocess.TimeoutExpired
       Assumes NO input from stdin at all'''
    if path:
        env = os.environ.copy()
        env["PATH"] =  path + ":" + env["PATH"]
    if shell:
        try:
            result = subprocess.run(command, capture_output=True, check=False,
                timeout=timeout, input="", shell=True, env=env if path else None)
        except TypeError:
            result = subprocess.run(command, stderr=subprocess.PIPE,
                stdout=subprocess.PIPE, check=False, timeout=timeout,
                input="", shell=True, env=env if path else None)

    else:
        try:
            result = subprocess.run(command.split(), capture_output=True,
                check=False, timeout=timeout, input="", env=env if path else None)
        except TypeError:
            result = subprocess.run(command.split(), stderr=subprocess.PIPE,
                stdout=subprocess.PIPE, check=False, timeout=timeout,
                input="", env=env if path else None)
    return result.returncode, result.stderr.decode("utf-8"), result.stdout.decode("utf-8")


def parse_positions(position_str):
    d = {}
    for position in position_str.replace('MySQL56/', '').split(','):
        gtid, offset_range = position.strip().split(':')
        d[gtid] = offset_range
    return d

def determine_errant_gtid(first_run, second_run):
    ''' Determine from two runs of ShardReplicationPositions for a
        keyspace/shard if there is likely to be an errant GTID'''
    first = {}
    second = {}
    for line in first_run.split('\n'):
        if len(line) == 0: continue
        tablet, _, _, tablet_type, _, _, _, _, repl_positions, lag = line.split(' ')
        first[tablet] = {}
        first[tablet]["tablet_type"] = tablet_type
        first[tablet]["repl_positions"] = parse_positions(repl_positions)
        first[tablet]["lag"] = int(lag)
    for line in second_run.split('\n'):
        if len(line) == 0: continue
        tablet, _, _, tablet_type, _, _, _, _, repl_positions, lag = line.split(' ')
        second[tablet] = {}
        second[tablet]["tablet_type"] = tablet_type
        second[tablet]["repl_positions"] = parse_positions(repl_positions)
        second[tablet]["lag"] = int(lag)

    do_gtid_checks(first, second)

def check_permutation(tablet1_repl_positions, tablet2_repl_positions):
    for server_gtid in tablet1_repl_positions:
        if server_gtid not in tablet2_repl_positions:
            return False
    return True

def get_master(first):
    for tablet in first:
        if first[tablet]['tablet_type'] == 'master':
            return tablet
    return None

def do_gtid_checks(first, second):
    # First, check for server GTIDs not present everywhere
    for perm in permutations(first.keys(), 2):
        first_tablet_positions = first[perm[0]]['repl_positions']
        second_tablet_positions = first[perm[1]]['repl_positions']
        if not check_permutation(first_tablet_positions, second_tablet_positions):
            print("Tablet %s and %s have non-overlapping GTID positions:\n%s" % (perm[0], perm[1], json.dumps(first, indent=4)))
            sys.exit(4)
    for perm in permutations(second.keys(), 2):
        first_tablet_positions = second[perm[0]]['repl_positions']
        second_tablet_positions = second[perm[1]]['repl_positions']
        if not check_permutation(first_tablet_positions, second_tablet_positions):
            print("Tablet %s and %s have non-overlapping GTID positions:\n%s" % (perm[0], perm[1], json.dumps(second, indent=4)))
            sys.exit(5)

    # Now, check for cases that did not move between runs, where the
    #  replica is ahead of the master
    combos_to_examine = []
    for tablet in first:
        for gtid in first[tablet]['repl_positions']:
            first_repl_offset = first[tablet]['repl_positions'][gtid]
            second_repl_offset = second[tablet]['repl_positions'][gtid]
            if first_repl_offset != second_repl_offset:
                # Moved ahead, no problem here
                continue
            combos_to_examine.append(gtid)
    combos_to_examine = list(set(combos_to_examine))

    master_tablet = get_master(first)
    for gtid in combos_to_examine:
        master_offset = int(first[master_tablet]['repl_positions'][gtid].split('-')[-1])
        for tablet in first:
            if tablet == master_tablet: continue
            replica_offset = int(first[tablet]['repl_positions'][gtid].split('-')[-1])
            if replica_offset > master_offset:
                print("Master tablet %s is behind replica tablet %s for server GTID %s" % (master_tablet, tablet, gtid))
                sys.exit(6)


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage:  %s <vtctld_host>:<vtctld_port> <keyspace>" % sys.argv[0])
        sys.exit(1)
    if not shutil.which('vtctlclient'):
        print("Cannot find vtctlclient in path, bailing.")
        sys.exit(2)

    vtctld_spec = sys.argv[1]
    keyspace    = sys.argv[2]

    rc, _, stdout = run_command("vtctlclient -action_timeout=5s -server %s FindAllShardsInKeyspace %s" % (vtctld_spec, keyspace), shell=True)
    if rc:
        print("Probable timeout enumerating all shards in keyspace %s, bailing" % keyspace)
        sys.exit(10)


    shards = json.loads(stdout).keys()
    for shard in shards:
        rc, _, first_run_output = run_command("vtctlclient -action_timeout=5s -server %s ShardReplicationPositions %s/%s" % (vtctld_spec, keyspace, shard), shell=True)
        if rc:
            print("Probable timeout fetching ShardReplicationPositions for %s/%s, bailing" % (keyspace, shard))
            sys.exit(11)
        time.sleep(1)
        rc, _, second_run_output = run_command("vtctlclient -action_timeout=5s -server %s ShardReplicationPositions %s/%s" % (vtctld_spec, keyspace, shard), shell=True)
        if rc:
            print("Probable timeout fetching ShardReplicationPositions for %s/%s, bailing" % (keyspace, shard))
            sys.exit(12)
        determine_errant_gtid(first_run_output, second_run_output)

# TODO:
#   * Distinguish between timeout and other issues (e.g. wrong keyspace)
#   * Add tests
