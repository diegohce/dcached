[![GitHub release](https://img.shields.io/github/release/diegohce/dcached.svg)](https://github.com/diegohce/dcached/releases/)
[![Github all releases](https://img.shields.io/github/downloads/diegohce/dcached/total.svg)](https://github.com/diegohce/dcached/releases/)
[![GPLv3 license](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://github.com/diegohce/dcached/blob/master/LICENSE)
[![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg)](https://github.com/diegohce/dcached/graphs/commit-activity)
[![HitCount](http://hits.dwyl.io/diegohce/dcached.svg)](http://hits.dwyl.io/diegohce/dcached)


dcached(1)

# NAME

*dcached* - Distributed and masterless cache cluster.

# SYNOPSIS

*dcached* 

# DESCRIPTION

*dcached* can run as a standalone node or as a cluster of nodes. 
Config file at /etc/dcached.conf defines *dcached* behaviour.


# CONFIG FILE

```
cache:
  ip: "" #leave blank
  port: "9009" #listen port for cache service.
  gc_freq: 3600 #seconds, cache subsystem garbage collector
  mode: "standalone" # "cluster"

siblings:
  address: "224.0.0.1:9999" #multicast group.
  beacon_freq: 2 #seconds
  max_datagram_size: 128 #bytes, hostname MUST fit here.
  ttl: 5 #seconds, time to expire siblings registrar.
  beacon_interface: "" #Network interface to use to send the multicast beacon
```

# FILES

/etc/dcached.conf

/etc/default/dcached

# AUTHORS

Maintained by Diego Cena <diego.cena@gmail.com>. Up-to-date sources and binaries
can be found at https://github.com/diegohce/dcached and bugs/issues 
can be submitted there.

