BTSyncInator
============

##Multiple BitTorrent Sync Daemon Manager.

###Usage:
    btsyncinator [--config path/to/config.file] [--private-key path/to/privatekey] [--debug]

###Example:
    btsyncinator --config .btsyncinator.conf --private-key ~/.ssh/id_rsa

###Example config file:
    # Configuration file for BTSyncInator:
    [btsync-tester]
    sshuserstring=root
    serveraddrstring=example.server.com:22
    daemonaddrstring=localhost:9999

    [reference name]
    sshuserstring=username
    serveraddrstring=example2.server.com:60
    daemonaddrstring=localhost:9999
