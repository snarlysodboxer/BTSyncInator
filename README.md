BTSyncInator
============

##Multiple BitTorrent Sync Daemon Manager.

###Usage:
    btsyncinator [--config path/to/config.file] [--debug]

###Example:
    btsyncinator --config .btsyncinator.conf

###Example config file:
    # Configuration file for BTSyncInator:
    # The "default" section is for general options,
    # the other sections each represent a running BTSync daemon.
    [default]
    privatekeyfilepath=/home/user/.ssh/id_rsa
    serveaddress=localhost:10000
    tlskeypath=
    tlscertpath=
    # set usetls to true and leave tlskeypath & tlscertpath blank to generate a self-signed certificate.
    usetls=true

    [btsync-tester]
    sshuserstring=root
    serveraddrstring=example.server.com:22
    daemonaddrstring=localhost:9999

    [reference name]
    sshuserstring=username
    serveraddrstring=example2.server.com:60
    daemonaddrstring=localhost:9999

Note: reference names for each daemon must be unique.
