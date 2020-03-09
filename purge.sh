#!/bin/bash

set -e

make purge

case $1 in
    all)
        echo "Purging all products from WC backend"
        ./wc-purge --mode=all
        ;;
    ftp-only)
        echo "Removing image assets from FTP server"

        ./wc-purge --mode=ftp-only
        ;;
    *)
        echo "Please specify dev or production"; exit
        ;;
esac