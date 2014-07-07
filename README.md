BTSyncInator
============

##Multiple BitTorrent Sync Daemon Manager.

###Usage:
    btsyncinator [--config path/to/config.file] [--debug] [--apiDebug]

###Example:
    btsyncinator --config .btsyncinator.conf

###Example config file:
    # Configuration file for BTSyncInator:
    # The "default" section is for general options,
    # all other sections represent a running BTSync daemon.
    [default]
    privatekeypath=/home/user/.ssh/id_rsa
    serveaddress=localhost:10000
    # set usetls to true and leave tlskeypath or tlscertpath blank to generate a self-signed certificate:
    usetls=true
    tlskeypath=
    tlscertpath=
    # To enable login authentication, set a digestpath:
    digestpath=.btsync-digest

    [btsync-tester]
    sshuserstring=userOne
    serveraddrstring=example.server.com:22
    daemonaddrstring=localhost:9999

    [reference name]
    sshuserstring=username
    serveraddrstring=192.168.0.123:60
    daemonaddrstring=localhost:9999

Note: reference names for each daemon must be unique.
